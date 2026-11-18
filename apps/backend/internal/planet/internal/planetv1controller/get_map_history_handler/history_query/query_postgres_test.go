package history_query_test

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras/postgres_era_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/postgres_ledger_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_map_history_handler/history_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db    *cppg.Postgres
	store *postgres_ledger_store.Store
}

const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

var (
	since = time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)
	until = since.Add(2 * time.Hour)
)

func (s *testSuite) SetupSuite() {
	s.db = cppgtest.StartServer(s.T()).OpenSchema(s.T(), "planet", migrations.FS)
	s.store = postgres_ledger_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.Require().NoError(eras.Reseed(s.T().Context(), s.db))
}

func (s *testSuite) record(events ...ledger.Event) {
	stored := make([]inmemory_ledger_storage.Stored, 0, len(events))
	for i, event := range events {
		entry, err := event.Entry()
		s.Require().NoError(err)
		entry.Stage = ledger.StageOf(event)
		stored = append(stored, inmemory_ledger_storage.Stored{Position: ledger.Position(i + 1), Entry: entry}) //nolint:gosec // a few events.
	}
	s.Require().NoError(s.store.Save(s.T().Context(), inmemory_ledger_storage.Changes{Entries: slices.Values(stored)}))
}

func (s *testSuite) TestTheHistoryReplaysTheLedgerFromItsFirstEventIntoAFrameEachStep() {
	s.record(
		ledger.Taking{Tile: 1, Scope: "203.0.113.7", Account: ada, Country: "fr", At: since.Add(-24 * time.Hour)},
		ledger.Taking{Tile: 2, Scope: "203.0.113.7", Account: ada, Country: "de", At: since.Add(30 * time.Minute)},
		ledger.Spreading{Tile: 2, Scope: "203.0.113.7", Account: ada, Country: "de", At: since.Add(30 * time.Minute), Impacts: []clicks.Impact{
			{Tile: 3, Owner: "fr", Outcome: clicks.Taken},
		}},
		ledger.Striking{Tile: 1, Scope: "203.0.113.7", Account: ada, Country: "de", Owner: "fr", Shields: 1, At: since.Add(70 * time.Minute)},
		ledger.Taking{Tile: 1, Scope: "203.0.113.7", Account: ada, Country: "jp", At: until},
	)

	history, err := history_query.NewPostgresQuery(s.db, postgres_era_store.New(s.db)).History(s.T().Context(), since, until, time.Hour)
	s.Require().NoError(err)

	s.True(proto.Equal(&planetv1.MapHistory{
		Frames: []*planetv1.MapFrame{
			{At: timestamppb.New(since), Owners: []*planetv1.TileOwner{{Tile: 1, CountryId: "fr"}}},
			{At: timestamppb.New(since.Add(time.Hour)), Owners: []*planetv1.TileOwner{
				{Tile: 2, CountryId: "de", SecondsIntoStep: 1800}, {Tile: 3, CountryId: "de", SecondsIntoStep: 1800},
			}},
			{At: timestamppb.New(until), Owners: []*planetv1.TileOwner{}},
		},
		Hits: []*planetv1.TileHits{{Tile: 1, Hits: 1}, {Tile: 2, Hits: 1}, {Tile: 3, Hits: 1}},
		Eras: worldOnEarth1,
	}, history), "%v", history)
}

func (s *testSuite) TestTheHistoryReadsNothingBeforeTheLastResetAtOrBeforeItsWindow() {
	ctx := s.T().Context()
	s.record(ledger.Taking{Tile: 2, Scope: "203.0.113.7", Account: ada, Country: "de", At: since.Add(-30 * time.Minute)})
	// A spread the ledger cannot replay, before the reset: reading it would fail the history.
	_, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO ledger_events (position, map, board, tile, scope, country, previous, taken_at, kind, payload)
		VALUES (0, 'world', 1, 4, '203.0.113.7', 'fr', '', $1, 'spread', '{"taken":"not a list"}')`, since.Add(-2*time.Hour))
	s.Require().NoError(err)
	_, err = s.db.On(ctx).ExecContext(ctx, `INSERT INTO map_resets (id, at) VALUES ('season-1', $1)`, since.Add(-time.Hour))
	s.Require().NoError(err)

	history, err := history_query.NewPostgresQuery(s.db, postgres_era_store.New(s.db)).History(ctx, since, until, time.Hour)
	s.Require().NoError(err)

	s.True(proto.Equal(&planetv1.MapHistory{
		Frames: []*planetv1.MapFrame{
			{At: timestamppb.New(since), Owners: []*planetv1.TileOwner{{Tile: 2, CountryId: "de"}}},
			{At: timestamppb.New(since.Add(time.Hour)), Owners: []*planetv1.TileOwner{}},
			{At: timestamppb.New(until), Owners: []*planetv1.TileOwner{}},
		},
		Hits: []*planetv1.TileHits{},
		Eras: worldOnEarth1,
	}, history), "%v", history)
}

func (s *testSuite) TestTheWorldsMapStandsStillOverAnArenaAndJoinsUpAfterIt() {
	ctx := s.T().Context()
	battle := clicks.MapID("battle").On(clicks.TestBoard)
	s.Require().NoError(eras.KeepRow(ctx, s.db, eras.Earth2))
	s.record(
		ledger.Taking{Tile: 1, Scope: "203.0.113.7", Country: "fr", At: since.Add(10 * time.Minute)},
		ledger.Staged{On: battle, Event: ledger.Taking{Tile: 1, Scope: "203.0.113.7", Country: "br", At: since.Add(30 * time.Minute)}},
		ledger.Staged{On: battle, Event: ledger.Taking{Tile: 2, Scope: "203.0.113.7", Country: "br", At: since.Add(40 * time.Minute)}},
		ledger.Taking{Tile: 3, Scope: "203.0.113.7", Country: "de", At: since.Add(80 * time.Minute)},
		ledger.Staged{
			On:    clicks.World.On(clicks.TestBoard + 1),
			Event: ledger.Taking{Tile: 4, Scope: "203.0.113.7", Country: "de", At: since.Add(90 * time.Minute)},
		},
	)

	history, err := history_query.NewPostgresQuery(s.db, postgres_era_store.New(s.db)).History(ctx, since, until, time.Hour)
	s.Require().NoError(err)

	s.True(proto.Equal(&planetv1.MapHistory{
		Frames: []*planetv1.MapFrame{
			{At: timestamppb.New(since), Owners: []*planetv1.TileOwner{}},
			{At: timestamppb.New(since.Add(time.Hour)), Owners: []*planetv1.TileOwner{{Tile: 1, CountryId: "fr", SecondsIntoStep: 600}}},
			{At: timestamppb.New(until), Owners: []*planetv1.TileOwner{{Tile: 3, CountryId: "de", SecondsIntoStep: 1200}}},
		},
		Hits: []*planetv1.TileHits{{Tile: 1, Hits: 1}, {Tile: 3, Hits: 1}},
		Eras: worldOnEarth1,
	}, history), "the battle's takes, and one on another board, are on no frame and no hit, as each names its map: %v", history)
}

func (s *testSuite) TestAHistoryAcrossTwoErasDrawsEachOnItsBoardWithTheCut() {
	ctx := s.T().Context()
	fixture := clicks.World.On(eras.Fixture.ID)
	switched := since.Add(30 * time.Minute)
	s.Require().NoError(eras.KeepFixture(ctx, s.db))
	s.Require().NoError(postgres_era_store.New(s.db).Start(ctx, fixture, switched))
	s.record(
		ledger.Taking{Tile: 1, Scope: "203.0.113.7", Country: "fr", At: since.Add(-time.Hour)},
		ledger.Taking{Tile: 2, Scope: "203.0.113.7", Country: "fr", At: since.Add(10 * time.Minute)},
		ledger.Staged{On: fixture, Event: ledger.Taking{Tile: 5, Scope: "203.0.113.7", Country: "de", At: since.Add(40 * time.Minute)}},
		ledger.Taking{Tile: 3, Scope: "203.0.113.7", Country: "it", At: since.Add(50 * time.Minute)},
		ledger.Staged{On: fixture, Event: ledger.Taking{Tile: 6, Scope: "203.0.113.7", Country: "jp", At: since.Add(70 * time.Minute)}},
	)

	history, err := history_query.NewPostgresQuery(s.db, postgres_era_store.New(s.db)).History(ctx, since, until, time.Hour)
	s.Require().NoError(err)

	s.True(proto.Equal(&planetv1.MapHistory{
		Frames: []*planetv1.MapFrame{
			{At: timestamppb.New(since), Owners: []*planetv1.TileOwner{{Tile: 1, CountryId: "fr"}}},
			{At: timestamppb.New(since.Add(time.Hour)), Owners: []*planetv1.TileOwner{{Tile: 5, CountryId: "de", SecondsIntoStep: 2400}}},
			{At: timestamppb.New(until), Owners: []*planetv1.TileOwner{{Tile: 6, CountryId: "jp", SecondsIntoStep: 600}}},
		},
		Hits: []*planetv1.TileHits{{Tile: 2, Hits: 1}, {Tile: 5, Hits: 1, Era: 1}, {Tile: 6, Hits: 1, Era: 1}},
		Eras: []*planetv1.BoardEra{
			{StartedAt: timestamppb.New(eras.Epoch), Board: "earth-1"},
			{StartedAt: timestamppb.New(switched), Board: "fixture-1"},
		},
	}, history), "the frame before the cut is earth-1's; the first after it holds every tile held on fixture-1, whole; an "+
		"earth-1 take after the cut is no longer the world's map, and each hit names its era: %v", history)
}

func (s *testSuite) TestAWindowInsideTheSecondEraOpensOnItsBoardAlone() {
	ctx := s.T().Context()
	fixture := clicks.World.On(eras.Fixture.ID)
	s.Require().NoError(eras.KeepFixture(ctx, s.db))
	s.Require().NoError(postgres_era_store.New(s.db).Start(ctx, fixture, since.Add(-time.Hour)))
	s.record(
		ledger.Taking{Tile: 1, Scope: "203.0.113.7", Country: "fr", At: since.Add(-2 * time.Hour)},
		ledger.Staged{On: fixture, Event: ledger.Taking{Tile: 5, Scope: "203.0.113.7", Country: "de", At: since.Add(-time.Minute)}},
	)

	history, err := history_query.NewPostgresQuery(s.db, postgres_era_store.New(s.db)).History(ctx, since, until, time.Hour)
	s.Require().NoError(err)

	s.True(proto.Equal(&planetv1.MapFrame{At: timestamppb.New(since), Owners: []*planetv1.TileOwner{{Tile: 5, CountryId: "de"}}},
		history.GetFrames()[0]), "earth-1's tile 1 is another era's: %v", history)
	s.True(proto.Equal(&planetv1.BoardEra{StartedAt: timestamppb.New(since.Add(-time.Hour)), Board: "fixture-1"},
		history.GetEras()[0]), "the era in force at the window's start, alone")
	s.Len(history.GetEras(), 1)
}

// The world's one era, as the migration that made eras keeps it.
var worldOnEarth1 = []*planetv1.BoardEra{{StartedAt: timestamppb.New(eras.Epoch), Board: "earth-1"}}
