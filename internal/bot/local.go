package bot

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/mc-botnet/mc-botnet-server/internal/logger"
	"io"
	"os/exec"
	"strconv"
	"syscall"

	"github.com/knadh/koanf/v2"
)

type LocalRunner struct {
	l    *log.Logger
	cmd  string
	args []string
}

func NewLocalRunner(conf *koanf.Koanf) *LocalRunner {
	return &LocalRunner{
		l:    logger.NewLogger("runner", log.InfoLevel),
		cmd:  conf.MustString("runner.local.cmd"),
		args: conf.MustStrings("runner.local.args"),
	}
}

func (r *LocalRunner) Start(_ context.Context, opts *RunnerOptions) (RunnerHandle, error) {
	r.l.Info("runner: starting")

	cmd := exec.Command(r.cmd, r.args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Env = toEnv(opts)

	stdout, err := cmd.StdoutPipe()

	if err != nil {
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	go pipeOutput(stdout, opts)
	go pipeOutput(stderr, opts)

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	return &localRunnerHandle{cmd, done}, nil
}

func pipeOutput(r io.ReadCloser, opts *RunnerOptions) {
	scanner := bufio.NewScanner(r)
	id := opts.ID.String()
	id = id[len(id)-6:]
	l := logger.NewLogger(fmt.Sprintf("bot %s", id), log.DebugLevel)
	for scanner.Scan() {
		l.Debug(scanner.Text())
	}
}

type localRunnerHandle struct {
	cmd  *exec.Cmd
	done <-chan error
}

func (l *localRunnerHandle) Stop(ctx context.Context) error {
	pgid, err := syscall.Getpgid(l.cmd.Process.Pid)
	if err != nil {
		return err
	}

	err = syscall.Kill(-pgid, syscall.SIGTERM)
	if err != nil {
		return err
	}

	select {
	case err = <-l.done:
		return err
	case <-ctx.Done():
		return errors.Join(ctx.Err(), syscall.Kill(-pgid, syscall.SIGKILL))
	}
}

func toEnv(opts *RunnerOptions) []string {
	pair := func(key, value string) string {
		return fmt.Sprintf("%s=%s", key, value)
	}

	env := []string{
		pair("BOT_ID", opts.ID.String()),
		pair("GRPC_HOST", opts.GRPCHost),
		pair("GRPC_PORT", strconv.Itoa(opts.GRPCPort)),
	}

	return env
}
