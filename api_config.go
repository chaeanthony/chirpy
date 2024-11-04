package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/chaeanthony/chirpy/internal/auth"
	"github.com/chaeanthony/chirpy/internal/database"
	"github.com/google/uuid"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db *database.Queries
	platform string
	jwtSecret string
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

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	Token 		string 		`json:"token"`
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
  type parameters struct {
		Email 		string 		`json:"email"`
		Password  string 		`json:"password"`
	}
	type response struct {
		User
	}

	params := parameters{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("failed to decode request. expected email, got: %v", err))
		return
	}

	pw, err := auth.HashPassword(params.Password)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("failed to hash password: %v", err))
	}

	user, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{Email: params.Email, HashedPassword: pw})
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

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email 			string 		`json:"email"`
		Password  	string 		`json:"password"`
	}

	type response struct {
		User
		Token 				string `json:"token"`
		RefreshToken 	string `json:"refresh_token"`
	}

	params := parameters{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("failed to decode request. expected email, got: %v", err))
		return
	}

	usr, err := cfg.db.GetUserByEmail(r.Context(), params.Email)
	if errors.Is(err, sql.ErrNoRows) {
		WriteError(w, http.StatusNotFound, errors.New("failed to find user"))
		return 
	} else if err != nil {
		WriteError(w, http.StatusInternalServerError, errors.New("failed to get user"))
		return 
	}

	if err := auth.CheckPasswordHash(params.Password, usr.HashedPassword); err != nil {
		WriteError(w, http.StatusUnauthorized, errors.New("incorrect email or password"))
		return 
	}
	
	jwt, err := auth.MakeJWT(usr.ID, cfg.jwtSecret, time.Hour)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("failed to create token: %v", err))
		return 
	}

	refresh_token, err := auth.MakeRefreshToken()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("failed to create refresh token: %v", err))
		return 
	}

	_, err = cfg.db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token: refresh_token,
		UserID: usr.ID,
		ExpiresAt: sql.NullTime{Valid: true, Time: time.Now().AddDate(0, 0, 60)},
	})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("failed to store refresh token: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, response{
		User: User{
			ID: usr.ID,
			CreatedAt: usr.CreatedAt,
			UpdatedAt: usr.UpdatedAt,
			Email: usr.Email,
		},
		Token: jwt,
		RefreshToken: refresh_token,
	},
	)
}