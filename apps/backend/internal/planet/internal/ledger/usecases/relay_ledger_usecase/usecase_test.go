package relay_ledger_usecase_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/relay_ledger_usecase"
)

const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

var at = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

type cursor struct {
	position ledger.Position
}

func (c *cursor) Relayed(context.Context) (ledger.Position, error) { return c.position, nil }

func (c *cursor) SaveRelayed(_ context.Context, position ledger.Position) error {
	c.position = position
	return nil
}

type borders map[uint32]string

func (b borders) CountryOf(tile uint32) string { return b[tile] }

type sender struct {
	fails int
	sent  []proto.Message
}

func (s *sender) Send(_ context.Context, events ...proto.Message) error {
	if s.fails > 0 {
		s.fails--
		return errors.New("redis is restarting")
	}
	s.sent = append(s.sent, events...)
	return nil
}

func flushed(t *testing.T, events ...ledger.Event) *inmemory_ledger_storage.Storage {
	t.Helper()

	storage := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence(),
		slog.New(slog.DiscardHandler))
	require.NoError(t, storage.Load(t.Context()))
	for _, event := range events {
		storage.Append(event)
	}
	require.NoError(t, storage.Flush(t.Context()))
	return storage
}

func TestEachFactOfTheLedgerIsToldWithItsPositionInOrder(t *testing.T) {
	storage := flushed(t,
		ledger.Taking{Tile: 1, Scope: "1.2.3.4", Account: ada, Country: "fr", Previous: "de", At: at},
		ledger.Taking{Tile: 2, Scope: "1.2.3.4", Country: "fr", At: at},
		ledger.Bombing{Scope: "1.2.3.4", Account: ada, At: at, Blast: clicks.Blast{
			Tile: 9, CountryID: "fr", Cleared: []uint32{8, 9}, Owners: []string{"de", "de"},
		}},
		ledger.Fortifying{
			Tile: 3, Landmass: 7, Ground: "is", Tiles: 60, Scope: "1.2.3.4", Account: ada, Country: "fr", At: at,
			Raised: []clicks.TileShields{{Tile: 3, Shields: 1}},
		},
		ledger.Shielding{Tile: 3, Scope: "1.2.3.4", Account: ada, Country: "fr", Shields: 2, At: at},
	)
	events := &sender{}
	told := &cursor{}

	n, err := relay_ledger_usecase.New(storage, told, borders{1: "es", 9: "de"}, events).Execute(t.Context())
	require.NoError(t, err)

	assert.Equal(t, 5, n)
	assert.Equal(t, ledger.Position(5), told.position)
	want := []proto.Message{
		&planetv1.TileTaken{
			AccountId: ada, TileId: 1, Country: "fr", TakenAt: timestamppb.New(at), Position: 0, Previous: "de", Ground: "es",
		},
		&planetv1.BombLanded{Country: "fr", TileId: 9, Ground: "de", Cleared: 2, LandedAt: timestamppb.New(at), AccountId: ada, Position: 2},
		&planetv1.Fortified{Country: "fr", LandmassId: 7, Tiles: 60, Ground: "is", FortifiedAt: timestamppb.New(at), AccountId: ada, Position: 3},
		&planetv1.ShieldPlaced{AccountId: ada, TileId: 3, Country: "fr", Shields: 2, PlacedAt: timestamppb.New(at), Position: 4},
	}
	require.Len(t, events.sent, len(want), "a take with no account tells nothing, a bomb takes no tile")
	for i := range want {
		assert.True(t, proto.Equal(want[i], events.sent[i]), "%v", events.sent[i])
	}
}

func TestEachTakeOfASpreadIsToldWithItsIndexInTheEvent(t *testing.T) {
	storage := flushed(t,
		ledger.Taking{Tile: 1, Scope: "1.2.3.4", Account: ada, Country: "fr", At: at},
		ledger.Spreading{Tile: 1, Scope: "1.2.3.4", Account: ada, Country: "fr", At: at, Impacts: []clicks.Impact{
			{Tile: 2, Owner: "de", Outcome: clicks.Taken},
			{Tile: 3, Outcome: clicks.Taken},
			{Tile: 4, Owner: "de", Outcome: clicks.Shielded, Shields: 1},
			{Tile: 5, Outcome: clicks.Taken},
		}},
	)
	events := &sender{}

	_, err := relay_ledger_usecase.New(storage, &cursor{}, borders{}, events).Execute(t.Context())
	require.NoError(t, err)

	type place struct {
		tile, index uint32
		position    uint64
	}
	told := make([]place, 0, len(events.sent))
	for _, fact := range events.sent {
		if taken, ok := fact.(*planetv1.TileTaken); ok {
			told = append(told, place{tile: taken.GetTileId(), index: taken.GetIndex(), position: taken.GetPosition()})
		}
	}
	assert.Equal(t, []place{
		{tile: 1, index: 0, position: 0},
		{tile: 2, index: 0, position: 1},
		{tile: 3, index: 1, position: 1},
		{tile: 5, index: 2, position: 1},
	}, told, "the click, then the spread's three takes in one event; a shield struck is no take")
}

func TestEachShieldStruckIsToldWithWhoseItWasAndItsIndexInTheEvent(t *testing.T) {
	storage := flushed(t,
		ledger.Striking{Tile: 1, Scope: "1.2.3.4", Account: ada, Country: "fr", Owner: "pw", Shields: 3, At: at},
		ledger.Spreading{Tile: 2, Scope: "1.2.3.4", Account: ada, Country: "fr", At: at, Impacts: []clicks.Impact{
			{Tile: 2, Owner: "pw", Outcome: clicks.Shielded, Shields: 1},
			{Tile: 3, Owner: "de", Outcome: clicks.Taken},
			{Tile: 4, Owner: "dz", Outcome: clicks.Shielded},
		}},
		ledger.Bombing{Scope: "1.2.3.4", At: at, Blast: clicks.Blast{
			Tile: 9, CountryID: "fr", Cleared: []uint32{9}, Owners: []string{"de"},
			Struck: []uint32{7, 8}, Left: []int{0, 4}, StruckOwners: []string{"pw", "fr"},
		}},
	)
	events := &sender{}

	_, err := relay_ledger_usecase.New(storage, &cursor{}, borders{}, events).Execute(t.Context())
	require.NoError(t, err)

	var struck []proto.Message
	for _, fact := range events.sent {
		if _, ok := fact.(*planetv1.ShieldStruck); ok {
			struck = append(struck, fact)
		}
	}
	want := []proto.Message{
		&planetv1.ShieldStruck{AccountId: ada, TileId: 1, Country: "fr", Owner: "pw", StruckAt: timestamppb.New(at), Position: 0},
		&planetv1.ShieldStruck{AccountId: ada, TileId: 2, Country: "fr", Owner: "pw", StruckAt: timestamppb.New(at), Position: 1},
		&planetv1.ShieldStruck{AccountId: ada, TileId: 4, Country: "fr", Owner: "dz", StruckAt: timestamppb.New(at), Position: 1, Index: 1},
		&planetv1.ShieldStruck{TileId: 7, Country: "fr", Owner: "pw", StruckAt: timestamppb.New(at), Position: 2},
		&planetv1.ShieldStruck{TileId: 8, Country: "fr", Owner: "fr", StruckAt: timestamppb.New(at), Position: 2, Index: 1},
	}
	require.Len(t, struck, len(want), "a click, a spread and a bomb each strike, whoever's shield it was")
	for i := range want {
		assert.True(t, proto.Equal(want[i], struck[i]), "%v", struck[i])
	}
}

func TestAFailedSendIsToldAgainNextTime(t *testing.T) {
	storage := flushed(t, ledger.Taking{Tile: 1, Scope: "1.2.3.4", Account: ada, Country: "fr", At: at})
	events := &sender{fails: 1}
	told := &cursor{}
	relay := relay_ledger_usecase.New(storage, told, borders{}, events)

	_, err := relay.Execute(t.Context())
	require.Error(t, err)
	assert.Equal(t, ledger.Position(0), told.position)

	_, err = relay.Execute(t.Context())
	require.NoError(t, err)
	assert.Len(t, events.sent, 1)
	assert.Equal(t, ledger.Position(1), told.position)
}

func TestNothingNewSendsNothing(t *testing.T) {
	storage := flushed(t, ledger.Taking{Tile: 1, Scope: "1.2.3.4", Account: ada, Country: "fr", At: at})
	events := &sender{}
	told := &cursor{position: 1}

	n, err := relay_ledger_usecase.New(storage, told, borders{}, events).Execute(t.Context())

	require.NoError(t, err)
	assert.Zero(t, n)
	assert.Empty(t, events.sent)
}
