package count_takes_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/takes"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/takes/inmemory_take_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/takes/usecases/count_takes_usecase"
)

type memoryFeed struct {
	start   takes.Position
	log     []takes.Take
	limit   int
	through takes.Position
	err     error
	onBatch []func()
}

func (f *memoryFeed) Start(context.Context) (takes.Position, error) {
	return f.start, f.err
}

func (f *memoryFeed) Batch(_ context.Context, from takes.Position) (takes.Batch, error) {
	if f.err != nil {
		return takes.Batch{}, f.err
	}
	if len(f.onBatch) > 0 {
		f.onBatch[0]()
		f.onBatch = f.onBatch[1:]
	}

	var list []takes.Take
	next := from
	for _, take := range f.log {
		if take.Position() >= from && len(list) < f.limit {
			list = append(list, take)
			next = take.Position() + 1
		}
	}
	if f.through > next && len(list) < f.limit {
		next = f.through
	}
	return takes.BatchOf(from, next, list) //nolint:wrapcheck // a fake.
}

var (
	ada    = players.AccountID{15: 1}
	monday = time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
)

func takesOf(account players.AccountID, count int) []takes.Take {
	log := make([]takes.Take, count)
	for i := range log {
		log[i] = takes.TakeOf(takes.Position(i), account, "fr", monday, false)
	}
	return log
}

type fixture struct {
	players *inmemory_player_store.Store
	store   *inmemory_take_store.Store
	feed    *memoryFeed
	useCase *count_takes_usecase.UseCase
}

func newFixture(feed *memoryFeed) fixture {
	players := inmemory_player_store.New()
	store := inmemory_take_store.New(players)
	return fixture{players: players, store: store, feed: feed, useCase: count_takes_usecase.New(feed, store)}
}

func (f fixture) tiles(t *testing.T, account players.AccountID) uint64 {
	t.Helper()

	stats, err := f.players.Stats(t.Context(), account)
	require.NoError(t, err)
	return stats.TilesTaken()
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

func TestWithNoPositionTheCountBeginsWhereTheFeedStartsOnTheStatsTheBusCounted(t *testing.T) {
	f := newFixture(&memoryFeed{start: 2, log: takesOf(ada, 5), limit: 10})
	require.NoError(t, f.players.RecordTake(t.Context(), ada, monday))
	require.NoError(t, f.players.RecordTake(t.Context(), ada, monday))

	out, err := f.useCase.Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, count_takes_usecase.Out{Began: true, From: 2, Takes: 3, Counted: []players.AccountID{ada}}, out)
	assert.Equal(t, uint64(5), f.tiles(t, ada), "the two takes the bus counted, then the three after")

	out, err = f.useCase.Execute(t.Context())
	require.NoError(t, err)
	assert.Equal(t, count_takes_usecase.Out{From: 5, CaughtUp: true}, out)
}

func TestTheCountReadsBatchByBatchUntilItIsCaughtUp(t *testing.T) {
	f := newFixture(&memoryFeed{log: takesOf(ada, 5), limit: 2})

	for _, want := range []count_takes_usecase.Out{
		{Began: true, From: 0, Takes: 2, Counted: []players.AccountID{ada}},
		{From: 2, Takes: 2, Counted: []players.AccountID{ada}},
		{From: 4, Takes: 1, Counted: []players.AccountID{ada}},
		{From: 5, CaughtUp: true},
	} {
		out, err := f.useCase.Execute(t.Context())
		require.NoError(t, err)
		assert.Equal(t, want, out)
	}
	assert.Equal(t, uint64(5), f.tiles(t, ada))
}

func TestEntriesThatAreNotTakesAreReadPastAndTheCountMovesOn(t *testing.T) {
	f := newFixture(&memoryFeed{log: []takes.Take{takes.TakeOf(1, ada, "fr", monday, false)}, limit: 10, through: 4})

	out, err := f.useCase.Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, count_takes_usecase.Out{Began: true, From: 0, Takes: 1, Counted: []players.AccountID{ada}}, out)
	position, err := f.store.Position(t.Context())
	require.NoError(t, err)
	assert.Equal(t, takes.Position(4), position, "past the entries of other kinds read after the take")

	out, err = f.useCase.Execute(t.Context())
	require.NoError(t, err)
	assert.Equal(t, count_takes_usecase.Out{From: 4, CaughtUp: true}, out)
}

func TestACountThatFailedIsReadAgainAndCountedOnce(t *testing.T) {
	f := newFixture(&memoryFeed{log: takesOf(ada, 4), limit: 2})
	_, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)

	failed := errors.New("postgres is down")
	f.store.FailWith(failed)
	_, err = f.useCase.Execute(t.Context())
	require.ErrorIs(t, err, failed)

	f.store.Heal()
	f.catchUp(t)

	assert.Equal(t, uint64(4), f.tiles(t, ada))
}

func TestARebuildBetweenTheReadAndTheCountIsNotCountedOver(t *testing.T) {
	f := newFixture(&memoryFeed{log: takesOf(ada, 4), limit: 2})
	_, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)
	f.feed.onBatch = []func(){func() {
		_, err := f.store.Rewind(t.Context())
		require.NoError(t, err)
	}}

	out, err := f.useCase.Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, count_takes_usecase.Out{From: 2}, out, "the batch read from 2 is dropped, and the next read starts over")
	assert.Zero(t, f.tiles(t, ada))
	f.catchUp(t)
	assert.Equal(t, uint64(4), f.tiles(t, ada))
}

func TestAFeedThatCannotBeReadBeginsAndCountsNothing(t *testing.T) {
	down := errors.New("planet is down")
	f := newFixture(&memoryFeed{log: takesOf(ada, 2), limit: 2, err: down})

	_, err := f.useCase.Execute(t.Context())

	require.ErrorIs(t, err, down)
	_, err = f.store.Position(t.Context())
	require.ErrorIs(t, err, takes.ErrNotStarted)
}
