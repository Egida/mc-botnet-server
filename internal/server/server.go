package server

import (
	"context"
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/knadh/koanf/v2"
	"github.com/mc-botnet/mc-botnet-server/internal/auth"
	"github.com/mc-botnet/mc-botnet-server/internal/bot"
	"github.com/mc-botnet/mc-botnet-server/internal/logger"
	"net/http"
)

type Server struct {
	l           *log.Logger
	manager     *bot.Manager
	authService *auth.Service

	httpServer *http.Server
}

func NewServer(conf *koanf.Koanf, manager *bot.Manager, authService *auth.Service) (*Server, error) {
	s := &Server{
		l:           logger.NewLogger("server", log.InfoLevel),
		manager:     manager,
		authService: authService,
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

func w(fn http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	for _, m := range middlewares {
		fn = m(fn)
	}
	return fn
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.l.Info("shutting down")

	return s.httpServer.Shutdown(ctx)
}

func (s *Server) Run() error {
	s.l.Info("starting", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}
