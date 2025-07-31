package main

import (
	"context"
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/knadh/koanf/v2"
	"github.com/mc-botnet/mc-botnet-server/internal/auth"
	"github.com/mc-botnet/mc-botnet-server/internal/config"
	"github.com/mc-botnet/mc-botnet-server/internal/database"
	"github.com/mc-botnet/mc-botnet-server/internal/logger"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mc-botnet/mc-botnet-server/internal/bot"
	"github.com/mc-botnet/mc-botnet-server/internal/rpc"

	"github.com/mc-botnet/mc-botnet-server/internal/server"
)

func provideRunner(conf *koanf.Koanf) (bot.Runner, func(), error) {
	switch typ := conf.MustString("runner.type"); typ {
	case "local":
		runner := bot.NewLocalRunner(conf)
		return runner, func() {}, nil
	case "kubernetes":
		runner, err := bot.NewKubernetesRunner(conf)
		return runner, func() { shutdown(runner.Stop) }, err
	default:
		return nil, nil, fmt.Errorf("invalid runner type: %s", typ)
	}
}

func run() error {
	// Default logger
	l := logger.NewLogger("main", log.InfoLevel)
	log.SetDefault(l)

	// Config
	conf, err := config.NewConfig()
	if err != nil {
		return err
	}

	// Database
	db, err := database.NewDatabase(conf)
	if err != nil {
		return err
	}
	defer db.Close()

	// Store
	store := database.NewSQLStore(db)

	// Auth service
	authService, err := auth.NewService(conf, store)
	if err != nil {
		return err
	}

	// Runner
	runner, cleanup, err := provideRunner(conf)
	if err != nil {
		return err
	}
	defer cleanup()

	// Acceptor
	acceptor := rpc.NewAcceptor(conf)
	defer shutdown(acceptor.Shutdown)

	// Manager
	manager := bot.NewManager(runner, acceptor)

	// HTTP Server
	s, err := server.NewServer(conf, manager, authService)
	if err != nil {
		return err
	}
	defer shutdown(s.Shutdown)

	// Cancellation
	done := make(chan os.Signal)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancelCause(context.Background())

	// Run services
	go func() { cancel(acceptor.Run()) }()
	go func() { cancel(s.Run()) }()

	// Wait until shutdown
	select {
	case <-done:
		log.Info("termination signal received")
	case <-ctx.Done():
		log.Error(ctx.Err())
	}

	// Shutdown
	log.Info("shutting down")
	return nil
}

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func shutdown(fn func(ctx context.Context) error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err := fn(ctx)
	if err != nil {
		log.Error(err.Error())
	}
	cancel()
}
