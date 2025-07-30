package main

import (
	"context"
	"github.com/charmbracelet/log"
	"github.com/mc-botnet/mc-botnet-server/internal/config"
	"github.com/mc-botnet/mc-botnet-server/internal/database"
	"github.com/mc-botnet/mc-botnet-server/internal/logger"
	"os/signal"
	"syscall"
	"time"

	"github.com/mc-botnet/mc-botnet-server/internal/bot"
	"github.com/mc-botnet/mc-botnet-server/internal/rpc"

	"github.com/mc-botnet/mc-botnet-server/internal/server"
)

func main() {
	l := logger.NewLogger("main", log.InfoLevel)
	log.SetDefault(l)

	conf, err := config.NewConfig()
	if err != nil {
		log.Fatal(err.Error())
	}

	_, err = database.NewDatabase(conf)
	if err != nil {
		log.Fatal(err.Error())
	}

	var runner bot.Runner
	switch typ := conf.MustString("runner.type"); typ {
	case "local":
		runner = bot.NewLocalRunner(conf)
	case "kubernetes":
		k, err := bot.NewKubernetesRunner(conf)
		if err != nil {
			log.Fatal(err.Error())
		}
		runner = k
		defer shutdown(k.Stop)
	default:
		log.Fatal("invalid runner type", "type", typ)
	}

	// Create the gRPC acceptor
	acceptor := rpc.NewAcceptor(conf)
	defer shutdown(acceptor.Shutdown)

	// Create the bot manager
	manager := bot.NewManager(runner, acceptor)

	// Create the HTTP server
	s, err := server.NewServer(conf, manager)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer shutdown(s.Shutdown)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	errorCtx, errorCancel := context.WithCancelCause(context.Background())

	go func() { errorCancel(s.Run()) }()
	go func() { errorCancel(acceptor.Run()) }()

	select {
	case <-ctx.Done():
		log.Info("termination signal received")
	case <-errorCtx.Done():
		log.Error(errorCtx.Err().Error())
	}

	log.Info("shutting down")
}

func shutdown(fn func(ctx context.Context) error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err := fn(ctx)
	if err != nil {
		log.Error(err.Error())
	}
	cancel()
}
