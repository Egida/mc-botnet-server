package server

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/mc-botnet/mc-botnet-server/internal/auth"
	"github.com/mc-botnet/mc-botnet-server/internal/database"
)

func (s *Server) getUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	u, err := s.store.FindUser(r.Context(), id)
	if errors.Is(err, database.ErrNotFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJson(w, u)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	id := auth.UserID(r)

	u, err := s.store.FindUser(r.Context(), id)
	if errors.Is(err, database.ErrNotFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJson(w, u)
}
