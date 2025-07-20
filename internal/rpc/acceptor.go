package rpc

import (
	"context"
	"github.com/google/uuid"
	"github.com/mc-botnet/mc-botnet-server/internal/rpc/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"log/slog"
	"net"
	"sync"
)

type BotClient struct {
	pb.BotClient

	conn *grpc.ClientConn
}

func (b *BotClient) Close() error {
	return b.conn.Close()
}

// Acceptor listens on a port for incoming connections from newly launched bots and establishes gRPC connections with
// the bots as servers.
type Acceptor struct {
	pb.UnimplementedAcceptorServer

	mu      sync.RWMutex
	pending map[string]chan *BotClient

	server *grpc.Server
}

func NewAcceptor() *Acceptor {
	return &Acceptor{pending: make(map[string]chan *BotClient)}
}

func (a *Acceptor) Run(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	a.server = grpc.NewServer()
	pb.RegisterAcceptorServer(a.server, a)

	slog.Info("starting gRPC acceptor", "addr", addr)
	return a.server.Serve(lis)
}

func (a *Acceptor) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		a.server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		a.server.Stop()
		return ctx.Err()
	}
}

func (a *Acceptor) Ready(ctx context.Context, request *pb.ReadyRequest) (*emptypb.Empty, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, "acceptor: no peer found")
	}

	conn, err := grpc.NewClient(p.Addr.String())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	client := pb.NewBotClient(conn)

	a.mu.RLock()
	defer a.mu.RUnlock()

	ch, ok := a.pending[request.Id]
	if !ok {
		return nil, status.Error(codes.PermissionDenied, "acceptor: bot wasn't requested")
	}
	ch <- &BotClient{client, conn}

	return new(emptypb.Empty), nil
}

func (a *Acceptor) WaitForBot(ctx context.Context, id uuid.UUID) (*BotClient, error) {
	ch := make(chan *BotClient, 1)

	a.mu.Lock()
	a.pending[id.String()] = ch
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		delete(a.pending, id.String())
		a.mu.Unlock()
	}()

	select {
	case b := <-ch:
		return b, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
