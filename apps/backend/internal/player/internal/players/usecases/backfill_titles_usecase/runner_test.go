package backfill_titles_usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/backfill_titles_usecase"
)

type countingExecutor struct {
	calls *int
}

func (c countingExecutor) Execute(context.Context) (backfill_titles_usecase.Backfill, error) {
	*c.calls++
	return backfill_titles_usecase.Backfill{}, nil
}

func TestTheRunnerBackfillsOnceAndReturns(t *testing.T) {
	calls := 0

	backfill_titles_usecase.NewRunner(countingExecutor{calls: &calls}).Run(t.Context())

	assert.Equal(t, 1, calls)
}
