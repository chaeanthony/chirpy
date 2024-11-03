package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

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
		UserID 	uuid.UUID `json:"user_id"`
	}

	type response struct {
		Chirp
	}

	params := parameters{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		WriteError(w, http.StatusInternalServerError, errors.New("failed to decode request"))
		return
	}

	chirp, err := cfg.db.CreateChirp(r.Context(), database.CreateChirpParams{Body: params.Body, UserID: params.UserID})
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
	chirps, err := cfg.db.GetChirps(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("failed to get chirps: %v", err))
		return 
	}
	
	resp := make([]Chirp, len(chirps))
	for i, chirp := range chirps {
		resp[i].ID = chirp.ID
		resp[i].CreatedAt = chirp.CreatedAt
		resp[i].UpdatedAt = chirp.UpdatedAt
		resp[i].Body = chirp.Body
		resp[i].UserID = chirp.UserID
	}

	WriteJSON(w, http.StatusOK, resp)
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