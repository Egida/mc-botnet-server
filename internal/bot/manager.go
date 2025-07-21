package bot

import (
	"context"
	"github.com/google/uuid"
	"github.com/mc-botnet/mc-botnet-server/internal/rpc"
	"log/slog"
)

type StartOptions struct {
	BotID uuid.UUID

	McHost     string
	McPort     int
	McUsername string
	McAuth     string
	McToken    string

	GRPCHost string
	GRPCPort int
}

type Bot struct {
	ID     uuid.UUID
	client rpc.BotClient
}

type Manager struct {
	runner   Runner
	acceptor *rpc.Acceptor
}

func NewManager(runner Runner, acceptor *rpc.Acceptor) *Manager {
	return &Manager{runner, acceptor}
}

func (m *Manager) StartBot(ctx context.Context) error {
	id := uuid.New()

	handle, err := m.runner.Start(ctx, &StartOptions{
		BotID:    id,
		GRPCHost: "localhost",
		GRPCPort: 8081,
	})
	if err != nil {
		return err
	}
	defer handle.Stop(ctx)
	slog.Info("started bot", "id", id)

	botClient, err := m.acceptor.WaitForBot(ctx, id)
	if err != nil {
		return err
	}
	slog.Info("connected to bot", "id", id)

	return botClient.Close()
}
