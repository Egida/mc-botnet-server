package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

func (s *Server) createBot(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	err := s.manager.StartBot(ctx)
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
