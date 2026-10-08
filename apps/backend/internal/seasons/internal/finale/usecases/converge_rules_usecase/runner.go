package converge_rules_usecase

import (
	"context"
	"time"
)

func NewRunner(every time.Duration, rules Executor, then Executor) *Runner {
	return &Runner{every: every, rules: rules, then: then}
}

// then runs only once the rules hold: the season's winner is read off a map already frozen.
type Runner struct {
	every time.Duration
	rules Executor
	then  Executor
}

func (r *Runner) Name() string { return "seasons-finale" }

// Waits one tick first: the runners start before the server listens.
func (r *Runner) Run(ctx context.Context) {
	ticker := time.NewTicker(r.every)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}

		if r.rules.Execute(ctx) == nil {
			_ = r.then.Execute(ctx)
		}
	}
}
