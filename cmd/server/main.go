package main

import (
	"context"
	"errors"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/mc-botnet/mc-botnet-server/internal/bot"
	"github.com/mc-botnet/mc-botnet-server/internal/rpc"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mc-botnet/mc-botnet-server/internal/server"
)

func main() {
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	go func() {
		err := acceptor.Run()
		if err != nil && !errors.Is(err, net.ErrClosed) {
			slog.Error(err.Error())
		}
		stop()
	}()

	go func() {
		err := s.Run()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error(err.Error())
		}
		stop()
	}()

	<-ctx.Done()

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
