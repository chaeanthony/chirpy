package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/chaeanthony/chirpy/internal/database"
	"github.com/google/uuid"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db *database.Queries
	platform string
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`
<html>

<body>
	<h1>Welcome, Chirpy Admin</h1>
	<p>Chirpy has been visited %d times!</p>
</body>

</html>
`, cfg.fileserverHits.Load())))
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		WriteError(w, http.StatusForbidden, errors.New("endpoint forbidden"))
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	cfg.fileserverHits.Store(0)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset to 0. Deleted all users."))
	if err := cfg.db.DeleteUsers(r.Context()); err != nil {
		WriteError(w, http.StatusInternalServerError, err)
	}
}

func (cfg *apiConfig) handlerValidateChirp(w http.ResponseWriter, r *http.Request) {
	profane := []string{"kerfuffle", "sharbert", "fornax"}
	cfg.fileserverHits.Add(1) 

	type parameters struct {
		Body string `json:"body"`
	}
	
	params := parameters{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		WriteError(w, http.StatusInternalServerError, errors.New("couldn't decode request body"))
	}
	
	if len(params.Body) > 140 {
		WriteError(w, http.StatusBadRequest, errors.New("invalid request"))
		return
	}

	words := strings.Split(params.Body, " ")
	for i, word := range words {
		for _, pw := range profane {
			if strings.ToLower(word) != pw {
				continue
			}
			words[i] = "****"
		}
	}

  WriteJSON(w, http.StatusOK, map[string]string{"cleaned_body": strings.Join(words, " ")})
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
  type parameters struct {
		Email string `json:"email"`
	}
	type response struct {
		User
	}

	params := parameters{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("failed to decode request. expected email, got: %v", err))
		return
	}

	user, err := cfg.db.CreateUser(r.Context(), params.Email)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, errors.New("failed to create user"))
		return 
	}

	WriteJSON(w, http.StatusCreated, response{
		User: User{
			ID: user.ID,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
			Email: user.Email,
		},
	})
}
