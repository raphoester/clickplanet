package count_takes_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/inmemory_contribution_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/count_takes_usecase"
)

type memoryFeed struct {
	start   standings.Position
	log     []standings.Entry
	limit   int
	through standings.Position
	err     error
	onBatch []func()
}

func (f *memoryFeed) Start(context.Context) (standings.Position, error) {
	return f.start, f.err
}

func (f *memoryFeed) Batch(_ context.Context, from standings.Position) (standings.Batch, error) {
	if f.err != nil {
		return standings.Batch{}, f.err
	}
	if len(f.onBatch) > 0 {
		f.onBatch[0]()
		f.onBatch = f.onBatch[1:]
	}

	var read []standings.Entry
	next := from
	for _, entry := range f.log {
		if entry.Position() >= from && len(read) < f.limit {
			read = append(read, entry)
			next = entry.Position() + 1
		}
	}
	if f.through > next && len(read) < f.limit {
		next = f.through
	}
	return standings.BatchOf(from, next, read) //nolint:wrapcheck // a fake.
}

var (
	ada  = standings.AccountID{15: 1}
	noon = time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)

	seasonZero = calendar.New(calendar.Config{List: []calendar.Entry{{Number: 0, EndsAt: noon.Add(24 * time.Hour), Finale: time.Hour}}})
)

func takesOf(count int) []standings.Entry {
	log := make([]standings.Entry, count)
	for i := range log {
		log[i] = standings.EntryOf(standings.Position(i), standings.Take{Account: ada, Country: "fr", At: noon}, false)
	}
	return log
}

type fixture struct {
	store   *inmemory_contribution_store.Store
	feed    *memoryFeed
	useCase *count_takes_usecase.UseCase
}

func newFixture(feed *memoryFeed) fixture {
	store := inmemory_contribution_store.New()
	return fixture{store: store, feed: feed, useCase: count_takes_usecase.New(feed, store, seasonZero)}
}

func (f fixture) tiles() uint64 {
	return f.store.Tally(0, ada).Tiles["fr"]
}

func (f fixture) catchUp(t *testing.T) {
	t.Helper()

	for range 100 {
		out, err := f.useCase.Execute(t.Context())
		require.NoError(t, err)
		if out.CaughtUp {
			return
		}
	}
	t.Fatal("never caught up")
}

func TestWithNoPositionTheCountBeginsWhereTheFeedStartsOnWhatTheBusCounted(t *testing.T) {
	f := newFixture(&memoryFeed{start: 2, log: takesOf(5), limit: 10})
	require.NoError(t, f.store.RecordTake(t.Context(), 0, standings.Take{Account: ada, Country: "fr", At: noon}))
	require.NoError(t, f.store.RecordTake(t.Context(), 0, standings.Take{Account: ada, Country: "fr", At: noon}))

	out, err := f.useCase.Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, count_takes_usecase.Out{Began: true, From: 2, Takes: 3}, out)
	assert.Equal(t, uint64(5), f.tiles(), "the two takes the bus counted, then the three after")
	f.catchUp(t)
	assert.Equal(t, uint64(5), f.tiles())
}

func TestEntriesThatAreNotTakesAreReadPastAndTheCountMovesOn(t *testing.T) {
	f := newFixture(&memoryFeed{log: takesOf(2), limit: 10, through: 6})

	f.catchUp(t)

	position, err := f.store.Position(t.Context())
	require.NoError(t, err)
	assert.Equal(t, standings.Position(6), position)
	assert.Equal(t, uint64(2), f.tiles())
}

func TestACountThatFailedIsReadAgainAndCountedOnce(t *testing.T) {
	f := newFixture(&memoryFeed{log: takesOf(4), limit: 2})
	_, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)

	failed := errors.New("postgres is down")
	f.store.FailWith(failed)
	_, err = f.useCase.Execute(t.Context())
	require.ErrorIs(t, err, failed)

	f.store.Heal()
	f.catchUp(t)

	assert.Equal(t, uint64(4), f.tiles())
}

func TestARebuildBetweenTheReadAndTheCountIsNotCountedOver(t *testing.T) {
	f := newFixture(&memoryFeed{log: takesOf(4), limit: 2})
	_, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)
	f.feed.onBatch = []func(){func() {
		_, err := f.store.Rewind(t.Context())
		require.NoError(t, err)
	}}

	out, err := f.useCase.Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, count_takes_usecase.Out{From: 2}, out)
	assert.Zero(t, f.tiles())
	f.catchUp(t)
	assert.Equal(t, uint64(4), f.tiles())
}

func TestAFeedThatCannotBeReadBeginsAndCountsNothing(t *testing.T) {
	down := errors.New("planet is down")
	f := newFixture(&memoryFeed{log: takesOf(2), limit: 2, err: down})

	_, err := f.useCase.Execute(t.Context())

	require.ErrorIs(t, err, down)
	_, err = f.store.Position(t.Context())
	require.ErrorIs(t, err, standings.ErrNotStarted)
}
