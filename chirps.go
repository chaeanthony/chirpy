package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/chaeanthony/chirpy/internal/auth"
	"github.com/chaeanthony/chirpy/internal/database"
	"github.com/google/uuid"
)

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time	`json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string		`json:"body"`
	UserID    uuid.UUID	`json:"user_id"`
}

func (cfg *apiConfig) handlerCreateChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body 		string 		`json:"body"`
		// UserID 	uuid.UUID `json:"user_id"`
	}

	type response struct {
		Chirp
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, fmt.Errorf("token required: %v", err))
		return 
	}
	userId, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, fmt.Errorf("invalid token: %v", err))
		return 
	}

	params := parameters{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		WriteError(w, http.StatusInternalServerError, errors.New("failed to decode request"))
		return
	}

	cleaned, err := validateChirp(params.Body)
	if err != nil {
		WriteError(w, http.StatusBadRequest, err)
		return
	}

	chirp, err := cfg.db.CreateChirp(r.Context(), database.CreateChirpParams{Body: cleaned, UserID: userId})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("failed to create chirp. got: %v", err))
		return 
	}

	WriteJSON(w, http.StatusCreated, response{
		Chirp: Chirp{
			ID: chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body: chirp.Body,
			UserID: chirp.UserID,
		},
	})
}

func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, r *http.Request) {
	dbChirps, err := cfg.db.GetChirps(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("couldn't retrieve chirps: %v", err))
		return
	}

	authorID := uuid.Nil
	authorIDString := r.URL.Query().Get("author_id")
	if authorIDString != "" {
		authorID, err = uuid.Parse(authorIDString)
		if err != nil {
			WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid author ID: %v", err))
			return
		}
	}

	chirps := []Chirp{}
	for _, dbChirp := range dbChirps {
		// IF there's an author Id, then skip all chirps by users who are not that author
		if authorID != uuid.Nil && dbChirp.UserID != authorID {
			continue
		}

		chirps = append(chirps, Chirp{
			ID:        dbChirp.ID,
			CreatedAt: dbChirp.CreatedAt,
			UpdatedAt: dbChirp.UpdatedAt,
			UserID:    dbChirp.UserID,
			Body:      dbChirp.Body,
		})
	}

	WriteJSON(w, http.StatusOK, chirps)
}

func (cfg *apiConfig) handlerGetChirp(w http.ResponseWriter, r *http.Request) {
	str := r.PathValue("chirpId")
	chirpId, err := uuid.Parse(str)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("failed to parse id: %v", err))
		return
	}

	chirp, err := cfg.db.GetChirpById(r.Context(), chirpId)
	if err != nil {
		WriteError(w, http.StatusNotFound, fmt.Errorf("failed to get chirp: %v", err))
		return 
	}

	WriteJSON(w, http.StatusOK, Chirp{
		ID: chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body: chirp.Body,
		UserID: chirp.UserID,
	})
}

func (cfg *apiConfig) handlerDeleteChirp(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, fmt.Errorf("token required: %v", err))
		return 
	}
	userId, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, fmt.Errorf("invalid token: %v", err))
		return 
	}

	str := r.PathValue("chirpId")
	chirpId, err := uuid.Parse(str)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("failed to parse chirp id: %v", err))
		return
	}

	chirp, err := cfg.db.GetChirpById(r.Context(), chirpId)
	if err != nil {
		WriteError(w, http.StatusNotFound, fmt.Errorf("failed to get chirp: %v", err))
		return 
	}
	if chirp.UserID != userId {
		WriteError(w, http.StatusForbidden, fmt.Errorf("incorrect chirp author"))
		return 
	}

	if err := cfg.db.DeleteChirp(r.Context(), chirpId); errors.Is(err, sql.ErrNoRows) {
		WriteError(w, http.StatusNotFound, fmt.Errorf("chirp not found. got: %v", err))
		return 
	} else if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("failed to delete chirp: %v", err))
		return 
	}

	WriteJSON(w, http.StatusNoContent, nil)
}

// helpers ---------------------------------------------------------
func validateChirp(body string) (string, error) {
	const maxChirpLength = 140
	if len(body) > maxChirpLength {
		return "", errors.New("Chirp is too long")
	}

	badWords := map[string]struct{}{
		"kerfuffle": {},
		"sharbert":  {},
		"fornax":    {},
	}
	cleaned := getCleanedBody(body, badWords)
	return cleaned, nil
}

func getCleanedBody(body string, badWords map[string]struct{}) string {
	words := strings.Split(body, " ")
	for i, word := range words {
		loweredWord := strings.ToLower(word)
		if _, ok := badWords[loweredWord]; ok {
			words[i] = "****"
		}
	}
	cleaned := strings.Join(words, " ")
	return cleaned
}
