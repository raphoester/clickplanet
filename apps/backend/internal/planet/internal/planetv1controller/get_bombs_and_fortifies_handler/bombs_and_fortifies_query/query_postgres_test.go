package bombs_and_fortifies_query_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_bombs_and_fortifies_handler/bombs_and_fortifies_query"
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
	cy  = "9e8d7c6b-5a49-4382-a1b0-c9d8e7f6a5b4"
)

var (
	at      = time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	borders = clicks.BordersOf(100, []uint32{20}, []uint32{30, 31, 32})
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

func (s *testSuite) asked(accounts ...string) *planetv1.GetBombsAndFortifiesResponse {
	ids := make([]ledger.AccountID, 0, len(accounts))
	for _, account := range accounts {
		id, err := ledger.AccountIDOf(account)
		s.Require().NoError(err)
		ids = append(ids, id)
	}
	answer, err := bombs_and_fortifies_query.NewPostgresQuery(s.db, borders).BombsAndFortifies(s.T().Context(), ids)
	s.Require().NoError(err)
	return answer
}

func fortify(account string, landmass clicks.LandmassID, tile uint32) ledger.Fortifying {
	return ledger.Fortifying{Tile: tile, Landmass: landmass, Scope: "203.0.113.7", Account: account, Country: "fr", At: at}
}

func bomb(account string) ledger.Bombing {
	return ledger.Bombing{Scope: "203.0.113.7", Account: account, At: at, Blast: clicks.Blast{Tile: 5, CountryID: "fr"}}
}

func (s *testSuite) TestEachAccountsBombsAndTheTilesOfEveryLandmassItFortified() {
	s.record(
		bomb(ada), bomb(ada), bomb(bob),
		fortify(ada, 2, 32), fortify(ada, 1, 20), fortify(ada, 1, 20),
		fortify(bob, 2, 30),
		ledger.Taking{Tile: 7, Scope: "203.0.113.7", Account: cy, Country: "fr", At: at},
		ledger.Bombing{Scope: "203.0.113.7", At: at, Blast: clicks.Blast{Tile: 6, CountryID: "fr"}},
	)

	s.True(proto.Equal(&planetv1.GetBombsAndFortifiesResponse{Accounts: []*planetv1.BombsAndFortifies{
		{AccountId: ada, BombsDropped: 2, TilesFortified: 5},
		{AccountId: bob, BombsDropped: 1, TilesFortified: 3},
	}}, s.asked(ada, bob, cy)), "%v", s.asked(ada, bob, cy))
}

func (s *testSuite) TestOnlyTheAccountsAskedAreAnswered() {
	s.record(bomb(ada), bomb(bob))

	s.True(proto.Equal(&planetv1.GetBombsAndFortifiesResponse{Accounts: []*planetv1.BombsAndFortifies{
		{AccountId: bob, BombsDropped: 1},
	}}, s.asked(bob)))
	s.Empty(s.asked().GetAccounts())
}
