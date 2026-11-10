package relay_ledger_usecase_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/relay_ledger_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
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

const tasmania clicks.LandmassID = 420

func (b borders) CountryOf(tile uint32) string { return b[tile] }

func (b borders) LandmassOf(tile uint32) clicks.LandmassID {
	if b[tile] == "" {
		return clicks.NoLandmass
	}
	return tasmania
}

type positions map[uint32]clicks.Vec3

func (p positions) Position(tile uint32) (clicks.Vec3, bool) {
	at, ok := p[tile]
	return at, ok
}

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

	n, err := relay_ledger_usecase.New(storage, told, borders{1: "es", 9: "au"}, positions{1: {X: 0.6, Z: 0.8}}, events).Execute(t.Context())
	require.NoError(t, err)

	assert.Equal(t, 5, n)
	assert.Equal(t, ledger.Position(5), told.position)
	want := []proto.Message{
		&planetv1.TilesTaken{AccountId: ada, Country: "fr", TakenAt: timestamppb.New(at), Position: 0, Tiles: []*planetv1.TakenTile{
			{TileId: 1, Previous: "de", Ground: "es", Point: &planetv1.GlobePoint{X: 0.6, Z: 0.8}},
		}},
		&planetv1.BombLanded{
			Country: "fr", TileId: 9, Ground: "au", LandmassId: uint32(tasmania), Cleared: 2, LandedAt: timestamppb.New(at), AccountId: ada, Position: 2,
		},
		&planetv1.Fortified{Country: "fr", LandmassId: 7, Tiles: 60, Ground: "is", FortifiedAt: timestamppb.New(at), AccountId: ada, Position: 3},
		&planetv1.ShieldPlaced{AccountId: ada, TileId: 3, Country: "fr", Shields: 2, PlacedAt: timestamppb.New(at), Position: 4},
	}
	require.Len(t, events.sent, len(want), "a take with no account tells nothing, a bomb takes no tile")
	for i := range want {
		assert.True(t, proto.Equal(want[i], events.sent[i]), "%v", events.sent[i])
	}
}

func TestTheTakesOfAnEventAreToldTogetherEachWithItsIndexInTheEvent(t *testing.T) {
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

	_, err := relay_ledger_usecase.New(storage, &cursor{}, borders{}, positions{}, events).Execute(t.Context())
	require.NoError(t, err)

	want := []proto.Message{
		&planetv1.TilesTaken{AccountId: ada, Country: "fr", TakenAt: timestamppb.New(at), Position: 0, Tiles: []*planetv1.TakenTile{
			{TileId: 1},
		}},
		&planetv1.TilesTaken{AccountId: ada, Country: "fr", TakenAt: timestamppb.New(at), Position: 1, Tiles: []*planetv1.TakenTile{
			{TileId: 2, Index: 0, Previous: "de"}, {TileId: 3, Index: 1}, {TileId: 5, Index: 2},
		}},
	}
	var taken []proto.Message
	for _, fact := range events.sent {
		if _, ok := fact.(*planetv1.TilesTaken); ok {
			taken = append(taken, fact)
		}
	}
	require.Len(t, taken, len(want), "the click, then the spread's three takes in one fact; a shield struck is no take")
	for i := range want {
		assert.True(t, proto.Equal(want[i], taken[i]), "%v", taken[i])
	}
}

func TestAnEnclosureOfMoreThanAPartIsToldInPartsWhoseIndexesFollowOn(t *testing.T) {
	impacts := make([]clicks.Impact, 0, relay_ledger_usecase.PartTiles+2)
	for tile := range uint32(relay_ledger_usecase.PartTiles + 2) {
		impacts = append(impacts, clicks.Impact{Tile: tile + 1, Owner: "de", Outcome: clicks.Taken})
	}
	storage := flushed(t, ledger.Enclosing{Tile: 9, Scope: "1.2.3.4", Account: ada, Country: "fr", At: at, Impacts: impacts})
	events := &sender{}

	_, err := relay_ledger_usecase.New(storage, &cursor{}, borders{}, positions{}, events).Execute(t.Context())
	require.NoError(t, err)

	require.Len(t, events.sent, 2, "a full part, then the rest")
	first, rest := events.sent[0].(*planetv1.TilesTaken), events.sent[1].(*planetv1.TilesTaken)
	require.Len(t, first.GetTiles(), relay_ledger_usecase.PartTiles)
	require.Len(t, rest.GetTiles(), 2)
	assert.Equal(t, first.GetPosition(), rest.GetPosition(), "both parts are the event's")
	assert.Equal(t, uint32(relay_ledger_usecase.PartTiles), rest.GetTiles()[0].GetIndex(), "the index runs on across the parts")
	assert.Equal(t, uint32(relay_ledger_usecase.PartTiles+1), rest.GetTiles()[1].GetIndex())
}

func TestTheShieldsAnEventStruckAreToldTogetherWithWhoseEachWas(t *testing.T) {
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

	_, err := relay_ledger_usecase.New(storage, &cursor{}, borders{}, positions{}, events).Execute(t.Context())
	require.NoError(t, err)

	var struck []proto.Message
	for _, fact := range events.sent {
		if _, ok := fact.(*planetv1.ShieldsStruck); ok {
			struck = append(struck, fact)
		}
	}
	want := []proto.Message{
		&planetv1.ShieldsStruck{AccountId: ada, Country: "fr", StruckAt: timestamppb.New(at), Position: 0, Tiles: []*planetv1.StruckTile{
			{TileId: 1, Owner: "pw"},
		}},
		&planetv1.ShieldsStruck{AccountId: ada, Country: "fr", StruckAt: timestamppb.New(at), Position: 1, Tiles: []*planetv1.StruckTile{
			{TileId: 2, Owner: "pw"}, {TileId: 4, Owner: "dz", Index: 1},
		}},
		&planetv1.ShieldsStruck{Country: "fr", StruckAt: timestamppb.New(at), Position: 2, Tiles: []*planetv1.StruckTile{
			{TileId: 7, Owner: "pw"}, {TileId: 8, Owner: "fr", Index: 1},
		}},
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
	relay := relay_ledger_usecase.New(storage, told, borders{}, positions{}, events)

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

	n, err := relay_ledger_usecase.New(storage, told, borders{}, positions{}, events).Execute(t.Context())

	require.NoError(t, err)
	assert.Zero(t, n)
	assert.Empty(t, events.sent)
}

func TestClicksForOneFlagOnOneTileAtOnceTellOneTake(t *testing.T) {
	for range 500 {
		tiles := inmemory_tile_storage.New(clicks.BordersOf(100), inmemory_tile_storage.Config{},
			inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}), cpatomic.FakeTransactioner{}, &inmemory_tile_storage.RecordedPlacements{}, cptime.SystemClock{}, slog.New(slog.DiscardHandler))
		require.NoError(t, tiles.Set(t.Context(), 7, "de"))
		storage := flushed(t)
		recording := ledger.NewRecording(tiles, clicks.NewClaiming(tiles, func() int { return 10 }), storage, cptime.NewFixedClock(at))
		ctx := cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), ada)

		start := make(chan struct{})
		var wg sync.WaitGroup
		for range 8 {
			wg.Go(func() {
				<-start
				_, err := recording.Click(ctx, 7, "fr")
				assert.NoError(t, err)
			})
		}
		close(start)
		wg.Wait()
		require.NoError(t, storage.Flush(t.Context()))

		events := &sender{}
		_, err := relay_ledger_usecase.New(storage, &cursor{}, borders{}, positions{}, events).Execute(t.Context())
		require.NoError(t, err)

		require.Len(t, events.sent, 1)
		require.True(t, proto.Equal(&planetv1.TilesTaken{
			AccountId: ada, Country: "fr", TakenAt: timestamppb.New(at), Tiles: []*planetv1.TakenTile{{TileId: 7, Previous: "de"}},
		}, events.sent[0]), "%v", events.sent[0])
	}
}
