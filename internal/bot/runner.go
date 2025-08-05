package bot

import (
	"context"
	"github.com/google/uuid"
)

type RunnerOptions struct {
	ID       uuid.UUID
	GRPCHost string
	GRPCPort int
}

type Runner interface {
	Start(ctx context.Context, opts *RunnerOptions) (RunnerHandle, error)
}

type RunnerHandle interface {
	Stop(ctx context.Context) error
}
