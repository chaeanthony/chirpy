package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/chaeanthony/chirpy/internal/auth"
	"github.com/chaeanthony/chirpy/internal/database"
	"github.com/google/uuid"
)

type User struct {
	ID        	uuid.UUID `json:"id"`
	CreatedAt 	time.Time `json:"created_at"`
	UpdatedAt 	time.Time `json:"updated_at"`
	Email     	string    `json:"email"`
	IsChirpyRed bool 			`json:"is_chirpy_red"`
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
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
			ID:        	user.ID,
			CreatedAt: 	user.CreatedAt,
			UpdatedAt: 	user.UpdatedAt,
			Email:     	user.Email,
			IsChirpyRed: user.IsChirpyRed,
		},
	})
}

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type response struct {
		User
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
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
		Token:     refresh_token,
		UserID:    usr.ID,
		ExpiresAt: sql.NullTime{Valid: true, Time: time.Now().AddDate(0, 0, 60)},
	})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("failed to store refresh token: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, response{
		User: User{
			ID:        		usr.ID,
			CreatedAt: 		usr.CreatedAt,
			UpdatedAt: 		usr.UpdatedAt,
			Email:     		usr.Email,
			IsChirpyRed: 	usr.IsChirpyRed,
		},
		Token:        	jwt,
		RefreshToken: 	refresh_token,
	},
	)
}

func (cfg *apiConfig) handlerRefreshToken(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, fmt.Errorf("token required: %v", err))
		return 
	}

	dbToken, err := cfg.db.GetToken(r.Context(), token)
	if errors.Is(err, sql.ErrNoRows) {
		WriteError(w, http.StatusUnauthorized, fmt.Errorf("unauthorized. token may be expired"))
		return
	} else if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("failed to search token in db: %v", err))
		return
	}
	if (dbToken.ExpiresAt.Valid && dbToken.ExpiresAt.Time.Before(time.Now()) || dbToken.RevokedAt.Valid) {
		WriteError(w, http.StatusUnauthorized, fmt.Errorf("unauthorized. expired token"))
		return
	}

	jwt, err := auth.MakeJWT(dbToken.UserID, cfg.jwtSecret, time.Hour)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("failed to create token: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"token": jwt})
}

func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, fmt.Errorf("token required: %v", err))
		return 
	}

	_, err = cfg.db.UpdateToken(r.Context(), 
	database.UpdateTokenParams{
		RevokedAt: sql.NullTime{Valid: true, Time: time.Now()}, 
		UpdatedAt: time.Now(), 
		Token: token,
	})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("failed to revoke token: %v", err))
		return 
	}

	WriteJSON(w, http.StatusNoContent, nil)
}

func (cfg *apiConfig) handlerUpdateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	type response struct {
		User
		Token string `json:"token"`
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

	// decode params to get new email/password from request
	params := parameters{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("failed to decode request. expected email, got: %v", err))
		return
	}

	pw, err := auth.HashPassword(params.Password)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("failed to hash password: %v", err))
		return
	}

	updated_user, err := cfg.db.UpdateUser(r.Context(), database.UpdateUserParams{ID: userId, Email: params.Email, HashedPassword: pw})
	if errors.Is(err, sql.ErrNoRows) {
		WriteError(w, http.StatusNotFound, errors.New("failed to find user"))
		return
	} else if err != nil {
		WriteError(w, http.StatusInternalServerError, errors.New("failed to update user"))
		return
	}

	WriteJSON(w, http.StatusOK, response{
		User: User{
			ID: updated_user.ID,
			CreatedAt: updated_user.CreatedAt,
			UpdatedAt: updated_user.UpdatedAt,
			Email: updated_user.Email,
			IsChirpyRed: updated_user.IsChirpyRed,
		},
	})
}