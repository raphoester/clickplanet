package take_snapshot_usecase

import (
	"context"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	// How often the runner asks whether the day's snapshot is due: a restart catches up within it.
	tick = 10 * time.Minute
	// How long a snapshot that failed waits before it is tried again.
	retryAfter = time.Hour
)

// When the newest snapshot was taken: the store.
type Latest interface {
	Latest(ctx context.Context) (time.Time, error)
}

func NewRunner(snapshot Executor, latest Latest, clock cptime.Clock, logger *slog.Logger) *Runner {
	return &Runner{snapshot: snapshot, latest: latest, clock: clock, logger: logger}
}

// The day's snapshot, once a day, in the server: none is missed because a job did not run, and a restart catches up.
type Runner struct {
	snapshot Executor
	latest   Latest
	clock    cptime.Clock
	logger   *slog.Logger
	// After a snapshot failed, none is tried before then.
	retryAt time.Time
}

func (r *Runner) Name() string {
	return "publishing snapshot"
}

func (r *Runner) Run(ctx context.Context) {
	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	for {
		r.Once(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// The snapshot taken if the day's is due and none was taken since; a failure is tried again an hour later.
func (r *Runner) Once(ctx context.Context) {
	now := r.clock.Now()
	if now.Before(r.retryAt) {
		return
	}
	latest, err := r.latest.Latest(ctx)
	if err != nil {
		r.logger.Error("the newest snapshot could not be read", slog.String("error", err.Error()))
		return
	}
	if !latest.Before(stats.DueSince(now)) {
		return
	}
	if _, err := r.snapshot.Execute(ctx); err != nil {
		r.retryAt = now.Add(retryAfter)
	}
}
