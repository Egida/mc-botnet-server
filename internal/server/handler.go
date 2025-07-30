package server

import (
	"net/http"
)

func (s *Server) createBot(w http.ResponseWriter, r *http.Request) {
	err := s.manager.StartBot(r.Context())
	if err != nil {
		s.l.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
