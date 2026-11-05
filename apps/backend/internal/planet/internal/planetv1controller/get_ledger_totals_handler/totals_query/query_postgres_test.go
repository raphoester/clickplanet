package totals_query_test

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/postgres_ledger_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_ledger_totals_handler/totals_query"
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

const (
	ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	bob = "5a0c2f1e-7d3b-4c8a-9e6f-1b2d3c4e5f60"
)

var (
	since = time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)
	until = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	at    = since.Add(time.Hour)
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

func (s *testSuite) totals() *planetv1.LedgerTotals {
	answer, err := totals_query.NewPostgresQuery(s.db).Totals(s.T().Context(), since, until)
	s.Require().NoError(err)
	return answer
}

func (s *testSuite) TestTheTotalsCountEachKindOfActAndTheAccountsThatMadeThem() {
	s.record(
		ledger.Taking{Tile: 1, Scope: "203.0.113.7", Account: ada, Country: "fr", Previous: "de", At: at},
		ledger.Taking{Tile: 2, Scope: "203.0.113.7", Account: ada, Country: "fr", At: at},
		ledger.Spreading{Tile: 3, Scope: "203.0.113.7", Account: ada, Country: "fr", At: at, Impacts: []clicks.Impact{
			{Tile: 3, Owner: "es", Outcome: clicks.Taken},
		}},
		ledger.Enclosing{Tile: 7, Scope: "203.0.113.7", Account: ada, Country: "it", At: at, Impacts: []clicks.Impact{
			{Tile: 7, Owner: "fr", Outcome: clicks.Taken},
		}},
		ledger.Bombing{Scope: "203.0.113.7", Account: ada, At: at, Blast: clicks.Blast{
			Tile: 9, CountryID: "fr", Cleared: []uint32{9, 10}, Owners: []string{"de", "de"},
		}},
		ledger.Striking{Tile: 11, Scope: "203.0.113.7", Account: ada, Country: "fr", Owner: "de", Shields: 1, At: at},
		ledger.Shielding{Tile: 12, Scope: "203.0.113.7", Account: bob, Country: "de", Shields: 3, At: at},
		ledger.Fortifying{
			Tile: 12, Landmass: 4, Scope: "203.0.113.7", Account: bob, Country: "de", At: at,
			Raised: []clicks.TileShields{{Tile: 12, Shields: 4}},
		},
		ledger.Taking{Tile: 14, Scope: "198.51.100.4", Country: "fr", Previous: "de", At: at},
	)

	s.True(proto.Equal(&planetv1.LedgerTotals{
		Takes: 3, Strikes: 1, Spreads: 1, Encloses: 1, Bombs: 1, Shields: 1, Fortifies: 1, Accounts: 2,
	}, s.totals()), "%v", s.totals())
}

func (s *testSuite) TestTheTotalsCountOnlyTheWindowFromItsStartToBeforeItsEnd() {
	s.record(
		ledger.Taking{Tile: 1, Scope: "203.0.113.7", Account: ada, Country: "fr", At: since.Add(-time.Second)},
		ledger.Taking{Tile: 2, Scope: "203.0.113.7", Account: ada, Country: "fr", At: since},
		ledger.Taking{Tile: 3, Scope: "203.0.113.7", Account: bob, Country: "de", At: until.Add(-time.Second)},
		ledger.Taking{Tile: 4, Scope: "203.0.113.7", Account: bob, Country: "de", At: until},
	)

	s.True(proto.Equal(&planetv1.LedgerTotals{Takes: 2, Accounts: 2}, s.totals()), "%v", s.totals())
}
