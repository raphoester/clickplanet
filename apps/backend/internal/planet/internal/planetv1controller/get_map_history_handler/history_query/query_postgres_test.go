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
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/postgres_ledger_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_map_history_handler/history_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
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
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "planet", migrations.FS)
	s.store = postgres_ledger_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) record(events ...ledger.Event) {
	stored := make([]inmemory_ledger_storage.Stored, 0, len(events))
	for i, event := range events {
		entry, err := event.Entry()
		s.Require().NoError(err)
		stored = append(stored, inmemory_ledger_storage.Stored{Position: ledger.Position(i), Entry: entry}) //nolint:gosec // a few events.
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

	history, err := history_query.NewPostgresQuery(s.db).History(s.T().Context(), since, until, time.Hour)
	s.Require().NoError(err)

	s.True(proto.Equal(&planetv1.MapHistory{
		Frames: []*planetv1.MapFrame{
			{At: timestamppb.New(since), Owners: []*planetv1.TileOwner{{Tile: 1, CountryId: "fr"}}},
			{At: timestamppb.New(since.Add(time.Hour)), Owners: []*planetv1.TileOwner{{Tile: 2, CountryId: "de"}, {Tile: 3, CountryId: "de"}}},
			{At: timestamppb.New(until), Owners: []*planetv1.TileOwner{}},
		},
		Hits: []*planetv1.TileHits{{Tile: 1, Hits: 1}, {Tile: 2, Hits: 1}, {Tile: 3, Hits: 1}},
	}, history), "%v", history)
}
