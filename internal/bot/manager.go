package bot

import (
	"context"
	"errors"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"github.com/mc-botnet/mc-botnet-server/internal/logger"
	"github.com/mc-botnet/mc-botnet-server/internal/model"
	"github.com/mc-botnet/mc-botnet-server/internal/rpc"
	"github.com/mc-botnet/mc-botnet-server/internal/rpc/pb"
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
	client *rpc.BotClient
	handle RunnerHandle
}

func (b *Bot) Stop(ctx context.Context) error {
	return errors.Join(b.client.Close(), b.handle.Stop(ctx))
}

type Manager struct {
	runner   Runner
	acceptor *rpc.Acceptor
	l        *log.Logger

	mu   sync.RWMutex
	bots map[uuid.UUID]*Bot
}

func NewManager(runner Runner, acceptor *rpc.Acceptor) *Manager {
	return &Manager{
		runner:   runner,
		acceptor: acceptor,
		l:        logger.NewLogger("manager", log.InfoLevel),
		bots:     make(map[uuid.UUID]*Bot),
	}
}

func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var wg sync.WaitGroup
	for _, bot := range m.bots {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := bot.Stop(ctx)
			if err != nil {
				m.l.Error("failed to stop bot", "err", err)
			}
		}()
	}
	wg.Wait()

	m.bots = make(map[uuid.UUID]*Bot)

	return nil
}

func (m *Manager) StartBot(ctx context.Context, req *model.StartBotRequest) error {
	id := uuid.New()

	handle, err := m.runner.Start(ctx, &RunnerOptions{
		ID:       id,
		GRPCHost: m.acceptor.Host(),
		GRPCPort: m.acceptor.Port(),
	})
	if err != nil {
		return err
	}
	m.l.Info("started bot", "id", id)

	client, err := m.acceptor.WaitForBot(ctx, id)
	if err != nil {
		return errors.Join(err, handle.Stop(ctx))
	}
	m.l.Info("connected to bot", "id", id)

	err = m.connectBot(ctx, client, req)
	if err != nil {
		return errors.Join(err, handle.Stop(ctx), client.Close())
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.bots[id] = &Bot{
		ID:     id,
		client: client,
		handle: handle,
	}

	return nil
}

func (m *Manager) connectBot(ctx context.Context, client *rpc.BotClient, req *model.StartBotRequest) error {
	// TODO implement online auth with credential storage
	_, err := client.Connect(ctx, &pb.ConnectRequest{
		Host: req.Host,
		Port: int32(req.Port),
		Auth: &pb.ConnectRequest_OfflineUsername{OfflineUsername: req.OfflineUsername},
	})
	return err
}
