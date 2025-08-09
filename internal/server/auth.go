package server

import (
	"errors"
	"net/http"

	"github.com/mc-botnet/mc-botnet-server/internal/auth"
	"github.com/mc-botnet/mc-botnet-server/internal/database"
	"github.com/mc-botnet/mc-botnet-server/internal/model"
)

func (s *Server) signUp(w http.ResponseWriter, r *http.Request) {
	body, ok := parseBody[model.SignUp](w, r)
	if !ok {
		return
	}

	token, err := s.authService.SignUp(r.Context(), body)
	if errors.Is(err, database.ErrConflict) {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJson(w, &model.AuthResponse{Token: token})
}

func (s *Server) signIn(w http.ResponseWriter, r *http.Request) {
	body, ok := parseBody[model.SignIn](w, r)
	if !ok {
		return
	}

	token, err := s.authService.SignIn(r.Context(), body)
	if errors.Is(err, database.ErrNotFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if errors.Is(err, auth.ErrUnauthorized) {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJson(w, &model.AuthResponse{Token: token})
}
