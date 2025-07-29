package rpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"sync"

	"github.com/google/uuid"
	"github.com/knadh/koanf/v2"
	"github.com/mc-botnet/mc-botnet-server/internal/rpc/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
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

	conf *koanf.Koanf

	mu      sync.Mutex
	pending map[string]chan *BotClient

	server *grpc.Server
}

func NewAcceptor(conf *koanf.Koanf) *Acceptor {
	return &Acceptor{conf: conf, pending: make(map[string]chan *BotClient)}
}

func (a *Acceptor) Run() error {
	addr := fmt.Sprintf(":%d", a.conf.MustInt("grpc.port"))

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	a.server = grpc.NewServer()
	pb.RegisterAcceptorServer(a.server, a)

	slog.Info("acceptor: starting", "addr", addr)
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
	slog.Debug("acceptor: /Ready")
	err := a.ready(ctx, request)
	if err != nil {
		slog.Error("acceptor: error in /Ready", "err", err)
		return nil, fmt.Errorf("acceptor: %w", err)
	}
	return new(emptypb.Empty), nil
}

func (a *Acceptor) ready(ctx context.Context, request *pb.ReadyRequest) error {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return status.Error(codes.Internal, "no peer found")
	}

	addr, err := clientAddr(p.Addr, request.Port)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	client := pb.NewBotClient(conn)

	a.mu.Lock()
	defer a.mu.Unlock()

	ch, ok := a.pending[request.Id]
	if !ok {
		return status.Error(codes.PermissionDenied, "bot wasn't requested")
	}

	ch <- &BotClient{client, conn}

	return nil
}

func clientAddr(addr net.Addr, port int32) (string, error) {
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return "", err
	}
	return net.JoinHostPort(host, strconv.Itoa(int(port))), nil
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
		// test connection
		_, err := b.Ping(ctx, new(emptypb.Empty))
		if err != nil {
			return nil, fmt.Errorf("acceptor: %w", err)
		}
		return b, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("acceptor: %w", ctx.Err())
	}
}
