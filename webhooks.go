package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUpgradeUserToRed(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Event string `json:"event"`
		Data  struct {
			UserID string `json:"user_id"`
		} `json:"data"`
	}

	params := parameters{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("failed to decode request. expected email, got: %v", err))
		return
	}

	if params.Event != "user.upgraded" { // ignore non upgraded events
		WriteJSON(w, http.StatusNoContent, nil)
		return 
	}

	userId, err := uuid.Parse(params.Data.UserID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("failed to parse userId"))
		return 
	}

	if err := cfg.db.UpgradeUserToChirpyRed(r.Context(), userId); errors.Is(err, sql.ErrNoRows) {
		WriteError(w, http.StatusNotFound, errors.New("failed to find user"))
		return
	} else if err != nil {
		WriteError(w, http.StatusInternalServerError, errors.New("failed to get user"))
		return
	}

	WriteJSON(w, http.StatusNoContent, nil)
}