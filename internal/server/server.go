package server

import (
	"context"
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/knadh/koanf/v2"
	"github.com/mc-botnet/mc-botnet-server/internal/bot"
	"github.com/mc-botnet/mc-botnet-server/internal/logger"
	"net/http"
)

type Server struct {
	conf       *koanf.Koanf
	manager    *bot.Manager
	l          *log.Logger
	httpServer *http.Server
}

func NewServer(conf *koanf.Koanf, manager *bot.Manager) (*Server, error) {
	s := &Server{
		conf:    conf,
		manager: manager,
		l:       logger.New("server", log.InfoLevel),
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
	s.l.Info("shutting down")

	return s.httpServer.Shutdown(ctx)
}

func (s *Server) Run() error {
	s.l.Info("starting", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}
