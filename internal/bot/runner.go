package bot

import (
	"context"
)

type Runner interface {
	Start(ctx context.Context, opts *StartOptions) (RunnerHandle, error)
}

type RunnerHandle interface {
	Stop(ctx context.Context) error
}
