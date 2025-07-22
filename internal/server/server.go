package server

import (
	"context"
	"fmt"
	"github.com/knadh/koanf/v2"
	"github.com/mc-botnet/mc-botnet-server/internal/bot"
	"log/slog"
	"net/http"
)

type Server struct {
	conf       *koanf.Koanf
	manager    *bot.Manager
	httpServer *http.Server
}

func NewServer(conf *koanf.Koanf, manager *bot.Manager) (*Server, error) {
	s := &Server{
		conf:    conf,
		manager: manager,
	}

	mux := registerRoutes(s)

	s.httpServer = &http.Server{
		Handler: mux,
		Addr:    fmt.Sprintf(":%d", conf.MustInt("server.port")),
	}

	return s, nil
}

func registerRoutes(s *Server) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /ping", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Pong!"))
	})

	mux.HandleFunc("POST /bot", s.createBot)

	return mux
}

func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("server: shutting down")

	return s.httpServer.Shutdown(ctx)
}

func (s *Server) Run() error {
	slog.Info("server: starting", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}
