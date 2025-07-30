package main

import (
	"context"
	"github.com/mc-botnet/mc-botnet-server/internal/config"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mc-botnet/mc-botnet-server/internal/bot"
	"github.com/mc-botnet/mc-botnet-server/internal/rpc"

	"github.com/mc-botnet/mc-botnet-server/internal/server"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	conf, err := config.NewConfig()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	var runner bot.Runner
	switch typ := conf.MustString("runner.type"); typ {
	case "local":
		runner = bot.NewLocalRunner(conf)
	case "kubernetes":
		k, err := bot.NewKubernetesRunner(conf)
		if err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}
		runner = k
		defer shutdown(k.Stop)
	default:
		slog.Error("invalid runner type", "type", typ)
		os.Exit(1)
	}

	// Create the gRPC acceptor
	acceptor := rpc.NewAcceptor(conf)
	defer shutdown(acceptor.Shutdown)

	// Create the bot manager
	manager := bot.NewManager(runner, acceptor)

	// Create the HTTP server
	s, err := server.NewServer(conf, manager)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	defer shutdown(s.Shutdown)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	errorCtx, errorCancel := context.WithCancelCause(context.Background())

	go func() { errorCancel(s.Run()) }()
	go func() { errorCancel(acceptor.Run()) }()

	select {
	case <-ctx.Done():
		slog.Info("termination signal received")
	case <-errorCtx.Done():
		slog.Error(errorCtx.Err().Error())
	}

	slog.Info("shutting down")
}

func shutdown(fn func(ctx context.Context) error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err := fn(ctx)
	if err != nil {
		slog.Error(err.Error())
	}
	cancel()
}
