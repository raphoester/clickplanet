package backfill_titles_usecase

import "context"

type Runner struct {
	executor Executor
}

func NewRunner(executor Executor) *Runner {
	return &Runner{executor: executor}
}

func (r *Runner) Name() string { return "player-titles-backfill" }

func (r *Runner) Run(ctx context.Context) {
	_, _ = r.executor.Execute(ctx)
}
