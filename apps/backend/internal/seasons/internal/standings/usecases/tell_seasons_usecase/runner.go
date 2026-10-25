package tell_seasons_usecase

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
)

type Runner struct {
	interval time.Duration
	executor Executor
}

func NewRunner(interval time.Duration, executor Executor) *Runner {
	return &Runner{interval: interval, executor: executor}
}

func (r *Runner) Name() string { return "seasons-tellings" }

func (r *Runner) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	var told []calendar.Number
	for {
		now, _ := r.executor.Execute(ctx, told)
		for _, season := range now {
			told = append(told, season.Season)
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}
