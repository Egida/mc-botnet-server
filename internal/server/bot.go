package server

import (
	"github.com/mc-botnet/mc-botnet-server/internal/model"
	"net/http"
)

func (s *Server) startBot(w http.ResponseWriter, r *http.Request) {
	body, ok := parseBody[model.StartBotRequest](w, r)
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
