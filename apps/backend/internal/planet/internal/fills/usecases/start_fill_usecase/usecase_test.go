package start_fill_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/fills"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/fills/usecases/start_fill_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var at = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

type ledger struct {
	steps *[]string
	err   error
}

func (l ledger) Flush(context.Context) error {
	*l.steps = append(*l.steps, "flush")
	return l.err
}

type runner struct {
	steps *[]string
}

func (r runner) Wake() {
	*r.steps = append(*r.steps, "wake")
}

func start(store *fills.MemoryStore, steps *[]string, flushing error) *start_fill_usecase.UseCase {
	return start_fill_usecase.New(store, ledger{steps: steps, err: flushing}, runner{steps: steps}, cptime.NewFixedClock(at))
}

func TestAFillIsKeptOnceTheLedgerIsFlushedThenTheRunnerIsWoken(t *testing.T) {
	store := fills.NewMemoryStore()
	var steps []string

	err := start(store, &steps, nil).Execute(t.Context(), start_fill_usecase.In{
		Request: "season-0:finale:fill", Source: fills.SourceInternal,
	})

	require.NoError(t, err)
	assert.Equal(t, []fills.Snapshot{{ID: "season-0:finale:fill", Source: fills.SourceInternal, StartedAt: at}}, store.Kept())
	assert.Equal(t, []string{"flush", "wake"}, steps)
}

func TestARequestSentAgainChangesNothing(t *testing.T) {
	store := fills.NewMemoryStore()
	running := fills.Snapshot{
		ID: "season-0:finale:fill", Source: fills.SourceInternal, StartedAt: at.Add(-time.Minute),
		Reached: "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11", Filled: 500,
	}
	store.Keep(running)
	var steps []string

	err := start(store, &steps, nil).Execute(t.Context(), start_fill_usecase.In{
		Request: "season-0:finale:fill", Source: fills.SourceOperator,
	})

	require.NoError(t, err)
	assert.Equal(t, []fills.Snapshot{running}, store.Kept(), "the fill in progress keeps its start and how far it got")
	assert.Empty(t, steps)
}

func TestAFillThatCannotStartKeepsNothing(t *testing.T) {
	down := errors.New("postgres is down")
	for _, tc := range []struct {
		request  bonuses.FillID
		flushing error
		want     error
	}{
		{request: "", want: fills.ErrNoRequest},
		{request: "a1b2", flushing: down, want: down},
	} {
		store := fills.NewMemoryStore()
		var steps []string

		err := start(store, &steps, tc.flushing).Execute(t.Context(), start_fill_usecase.In{
			Request: tc.request, Source: fills.SourceOperator,
		})

		require.ErrorIs(t, err, tc.want)
		assert.Empty(t, store.Kept())
		assert.NotContains(t, steps, "wake")
	}
}
