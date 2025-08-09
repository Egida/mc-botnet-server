package server

import (
	"net/http"

	"github.com/mc-botnet/mc-botnet-server/internal/model"
)

func (s *Server) startBot(w http.ResponseWriter, r *http.Request) {
	body, ok := parseBody[model.StartBotRequest](s, w, r)
	if !ok {
		return
	}

	err := s.manager.StartBot(r.Context(), body)
	if err != nil {
		s.l.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
