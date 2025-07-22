package bot

import (
	"context"
	"github.com/google/uuid"
	"github.com/mc-botnet/mc-botnet-server/internal/rpc"
	"log/slog"
	"sync"
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
	handle RunnerHandle
}

type Manager struct {
	runner   Runner
	acceptor *rpc.Acceptor

	mu   sync.RWMutex
	bots map[uuid.UUID]*Bot
}

func NewManager(runner Runner, acceptor *rpc.Acceptor) *Manager {
	return &Manager{
		runner:   runner,
		acceptor: acceptor,
		bots:     make(map[uuid.UUID]*Bot),
	}
}

func (m *Manager) Stop(ctx context.Context) error {
	var wg sync.WaitGroup
	for _, bot := range m.bots {
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := bot.client.Close()
			if err != nil {
				slog.Error("manager: failed to close client", "error", err, "id", bot.ID)
			}
			err = bot.handle.Stop(ctx)
			if err != nil {
				slog.Error("manager: failed to stop runner", "error", err, "id", bot.ID)
			}
		}()
	}
	wg.Wait()
	return nil
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
