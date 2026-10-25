package recount_stats_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/recount_stats_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

var now = time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)

func account(i int) players.AccountID {
	return players.AccountID{14: byte(i >> 8), 15: byte(i)}
}

type fakeLedger struct {
	kept  map[players.AccountID]players.Recount
	pages [][]players.AccountID
	err   error
}

func (f *fakeLedger) Recounts(_ context.Context, accounts []players.AccountID) (map[players.AccountID]players.Recount, error) {
	f.pages = append(f.pages, accounts)
	recounts := map[players.AccountID]players.Recount{}
	for _, account := range accounts {
		if recount, ok := f.kept[account]; ok {
			recounts[account] = recount
		}
	}
	return recounts, f.err
}

func played(t *testing.T, store *inmemory_player_store.Store, accounts ...int) {
	t.Helper()

	for _, i := range accounts {
		require.NoError(t, store.Record(t.Context(), account(i), players.TakeAt(now)))
	}
}

func stats(t *testing.T, store *inmemory_player_store.Store, i int) players.StatsSnapshot {
	t.Helper()

	kept, err := store.Stats(t.Context(), account(i))
	require.NoError(t, err)
	return kept.TakeSnapshot()
}

func TestEachAccountsBombsAndTilesFortifiedAreWhatTheLedgerHolds(t *testing.T) {
	store := inmemory_player_store.New()
	played(t, store, 1, 2, 3)
	require.NoError(t, store.Record(t.Context(), account(2), players.Stats.WithBomb))
	require.NoError(t, store.Record(t.Context(), account(3), players.Stats.WithBonus))
	ledger := &fakeLedger{kept: map[players.AccountID]players.Recount{
		account(1): {BombsDropped: 4, TilesFortified: 140},
		account(2): {BombsDropped: 1},
	}}
	events := cpbootstrap.NewRecordedEvents()

	recounted, err := recount_stats_usecase.New(store, ledger, events).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, 1, recounted, "the second account already held what the ledger holds, and the third holds nothing")
	assert.Equal(t, uint64(4), stats(t, store, 1).BombsDropped)
	assert.Equal(t, uint64(140), stats(t, store, 1).TilesFortified)
	assert.Equal(t, uint64(1), stats(t, store, 1).TilesTaken, "a recount leaves the other counters alone")
	assert.Equal(t, uint64(1), stats(t, store, 3).BonusesClaimed)
	require.Len(t, events.Published(), 1)
	assert.True(t, proto.Equal(&playerv1.StatsChanged{AccountId: account(1).String()}, events.Published()[0]),
		"a recount tells the titles, as a live count does")
}

func TestASecondRecountChangesNothing(t *testing.T) {
	store := inmemory_player_store.New()
	played(t, store, 1)
	ledger := &fakeLedger{kept: map[players.AccountID]players.Recount{account(1): {BombsDropped: 2}}}
	useCase := recount_stats_usecase.New(store, ledger, cpbootstrap.NewRecordedEvents())

	_, err := useCase.Execute(t.Context())
	require.NoError(t, err)
	recounted, err := useCase.Execute(t.Context())

	require.NoError(t, err)
	assert.Zero(t, recounted)
	assert.Equal(t, uint64(2), stats(t, store, 1).BombsDropped)
}

func TestTheLedgerIsAskedAPageAtATime(t *testing.T) {
	store := inmemory_player_store.New()
	for i := range 501 {
		played(t, store, i+1)
	}
	ledger := &fakeLedger{kept: map[players.AccountID]players.Recount{account(501): {BombsDropped: 1}}}

	recounted, err := recount_stats_usecase.New(store, ledger, cpbootstrap.NewRecordedEvents()).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, 1, recounted)
	require.Len(t, ledger.pages, 2)
	assert.Len(t, ledger.pages[0], 500)
	assert.Equal(t, []players.AccountID{account(501)}, ledger.pages[1])
}

func TestALedgerThatCannotBeAskedChangesNothing(t *testing.T) {
	store := inmemory_player_store.New()
	played(t, store, 1)
	cause := errors.New("planet is down")
	ledger := &fakeLedger{kept: map[players.AccountID]players.Recount{account(1): {BombsDropped: 2}}, err: cause}

	_, err := recount_stats_usecase.New(store, ledger, cpbootstrap.NewRecordedEvents()).Execute(t.Context())

	require.ErrorIs(t, err, cause)
	assert.Zero(t, stats(t, store, 1).BombsDropped)
}
