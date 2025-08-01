package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/knadh/koanf/v2"
	"github.com/mc-botnet/mc-botnet-server/internal/auth"
	"github.com/mc-botnet/mc-botnet-server/internal/bot"
	"github.com/mc-botnet/mc-botnet-server/internal/database"
	"github.com/mc-botnet/mc-botnet-server/internal/logger"
	"net/http"
)

type Server struct {
	l           *log.Logger
	manager     *bot.Manager
	authService *auth.Service
	store       database.Store

	httpServer *http.Server
}

func NewServer(
	conf *koanf.Koanf,
	manager *bot.Manager,
	authService *auth.Service,
	store database.Store,
) (*Server, error) {
	s := &Server{
		l:           logger.NewLogger("server", log.InfoLevel),
		manager:     manager,
		authService: authService,
		store:       store,
	}

	mux := router(s)

	s.httpServer = &http.Server{
		Handler: mux,
		Addr:    fmt.Sprintf(":%d", conf.MustInt("server.port")),
	}

	return s, nil
}

func router(s *Server) http.Handler {
	mux := http.NewServeMux()

	// Middlewares
	a := s.authService.Middleware

	mux.HandleFunc("GET /ping", ping)

	mux.HandleFunc("POST /bot", s.createBot)

	mux.HandleFunc("POST /auth/signup", s.signUp)
	mux.HandleFunc("POST /auth/signin", s.signIn)

	mux.HandleFunc("GET /user/{id}", a(s.getUser))
	mux.HandleFunc("GET /user/me", a(s.me))

	return mux
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) Run() error {
	s.l.Info("starting", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func parseBody[T any](w http.ResponseWriter, r *http.Request) (*T, bool) {
	var v T

	err := json.NewDecoder(r.Body).Decode(&v)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, false
	}

	return &v, true
}

func writeJson(w http.ResponseWriter, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Error("error marshaling response", "err", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

func ping(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("Pong!"))
}
