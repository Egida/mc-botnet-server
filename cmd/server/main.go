package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/mc-botnet/mc-botnet-server/internal/bot"
	"github.com/mc-botnet/mc-botnet-server/internal/rpc"

	"github.com/mc-botnet/mc-botnet-server/internal/server"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	conf := koanf.New(".")
	err := conf.Load(file.Provider("config.toml"), toml.Parser())
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	// Create the bot runner
	// runner, err := bot.NewKubernetesRunner(conf)
	// if err != nil {
	// 	slog.Error(err.Error())
	// 	os.Exit(1)
	// }

	runner := bot.NewLocalRunner(conf)

	// Create the gRPC acceptor
	acceptor := rpc.NewAcceptor(conf)

	// Create the bot manager
	manager := bot.NewManager(runner, acceptor)

	// Create the HTTP server
	s, err := server.NewServer(conf, manager)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
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

	slog.Info("shutting down gracefully")
	shutdownMany(s.Shutdown, acceptor.Shutdown /*, runner.Stop*/)
}

func shutdownMany(fns ...func(ctx context.Context) error) {
	for _, fn := range fns {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := fn(ctx)
		if err != nil {
			slog.Error(err.Error())
		}
		cancel()
	}
}
