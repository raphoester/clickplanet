package count_takes_usecase

import (
	"context"
	"time"
)

type Config struct {
	PollInterval time.Duration
}

const defaultPollInterval = time.Second

func (c Config) WithDefaults() Config {
	if c.PollInterval <= 0 {
		c.PollInterval = defaultPollInterval
	}
	return c
}

type Runner struct {
	interval time.Duration
	executor Executor
}

func NewRunner(config Config, executor Executor) *Runner {
	return &Runner{interval: config.WithDefaults().PollInterval, executor: executor}
}

func (r *Runner) Name() string { return "seasons-standings-takes" }

func (r *Runner) Run(ctx context.Context) {
	for {
		out, err := r.executor.Execute(ctx)
		if ctx.Err() != nil {
			return
		}
		if err == nil && !out.CaughtUp {
			continue
		}

		select {
		case <-time.After(r.interval):
		case <-ctx.Done():
			return
		}
	}
}
