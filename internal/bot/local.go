package bot

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strconv"
	"syscall"

	"github.com/knadh/koanf/v2"
)

type LocalRunner struct {
	conf *koanf.Koanf
}

func NewLocalRunner(conf *koanf.Koanf) *LocalRunner {
	return &LocalRunner{conf}
}

func (r *LocalRunner) Start(_ context.Context, opts *StartOptions) (RunnerHandle, error) {
	slog.Info("runner: starting")

	cmd := exec.Command(r.conf.MustString("bot.local.exec"), r.conf.MustStrings("bot.local.args")...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	opts.GRPCHost = "localhost"
	opts.GRPCPort = r.conf.MustInt("grpc.port")
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

func pipeOutput(r io.ReadCloser, opts *StartOptions) {
	scanner := bufio.NewScanner(r)
	id := opts.BotID.String()
	id = id[len(id)-6:]
	for scanner.Scan() {
		slog.Debug(fmt.Sprintf("bot %s: %s", id, scanner.Text()))
	}
}

type localRunnerHandle struct {
	cmd  *exec.Cmd
	done <-chan error
}

func (l *localRunnerHandle) Stop(ctx context.Context) error {
	slog.Info("runner: stopping")

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

func toEnv(opts *StartOptions) []string {
	pair := func(key, value string) string {
		return fmt.Sprintf("%s=%s", key, value)
	}

	env := []string{
		pair("BOT_ID", opts.BotID.String()),
		pair("BOT_HOST", opts.McHost),
		pair("BOT_PORT", strconv.Itoa(opts.McPort)),
		pair("BOT_USERNAME", opts.McUsername),
		pair("BOT_AUTH", opts.McAuth),
		pair("GRPC_HOST", opts.GRPCHost),
		pair("GRPC_PORT", strconv.Itoa(opts.GRPCPort)),
	}

	if opts.McToken != "" {
		env = append(env, pair("BOT_TOKEN", opts.McToken))
	}

	return env
}
