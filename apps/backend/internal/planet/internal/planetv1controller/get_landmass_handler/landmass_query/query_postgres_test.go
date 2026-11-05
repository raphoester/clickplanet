package landmass_query_test

import (
	"context"
	"log/slog"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/postgres_ledger_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_landmass_handler/landmass_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_landmass_handler/landmass_query/inmemory_tile_holders"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db    *cppg.Postgres
	store *postgres_ledger_store.Store
	tiles *inmemory_tile_storage.Storage
}

const (
	ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	bob = "5a0c2f1e-7d3b-4c8a-9e6f-1b2d3c4e5f60"
)

var at = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "planet", migrations.FS)
	s.store = postgres_ledger_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.tiles = inmemory_tile_storage.New(clicks.BordersOf(10, []uint32{1, 2, 3}, []uint32{4}), inmemory_tile_storage.Config{},
		inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}), cptime.SystemClock{}, slog.New(slog.DiscardHandler))
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

func (s *testSuite) take(account string, tile uint32, flag string) {
	s.Require().NoError(s.tiles.Click(cpctx.AddAccountToContext(context.Background(), account), tile, flag))
}

func (s *testSuite) landmass(landmass clicks.LandmassID) *planetv1.GetLandmassResponse {
	answer, err := landmass_query.NewPostgresQuery(s.db, inmemory_tile_holders.New(s.tiles)).Landmass(s.T().Context(), landmass)
	s.Require().NoError(err)
	return answer
}

func fortify(landmass clicks.LandmassID, account, flag string, minute int) ledger.Fortifying {
	return ledger.Fortifying{
		Tile: 1, Landmass: landmass, Ground: "it", Tiles: 3, Scope: "203.0.113.7", Account: account, Country: flag,
		At: at.Add(time.Duration(minute) * time.Minute),
	}
}

func (s *testSuite) TestTheLastFortifyOfTheLandmassIsTheLatestOfItsOwn() {
	s.record(fortify(1, ada, "fr", 0), fortify(1, bob, "de", 5), fortify(2, ada, "fr", 9))

	s.True(proto.Equal(&planetv1.LandmassFortification{
		AccountId: bob, CountryId: "de", FortifiedAtUnixMs: at.Add(5 * time.Minute).UnixMilli(),
	}, s.landmass(1).GetLastFortification()))
}

func (s *testSuite) TestALandmassNobodyFortifiedHasNoLastFortify() {
	s.record(fortify(2, ada, "fr", 0))

	s.Nil(s.landmass(1).GetLastFortification())
}

func (s *testSuite) TestAFortifyWhoseAccountWasDeletedNamesNobody() {
	s.record(fortify(1, "", "fr", 0))

	s.Empty(s.landmass(1).GetLastFortification().GetAccountId())
	s.Equal("fr", s.landmass(1).GetLastFortification().GetCountryId())
}

func (s *testSuite) TestTheHoldersAreTheTilesOfTheLandmassTheirTakesHold() {
	s.take(ada, 1, "fr")
	s.take(ada, 2, "fr")
	s.take(bob, 3, "de")
	s.take(bob, 4, "de")

	holders := s.landmass(1).GetHolders()

	s.Require().Len(holders, 2)
	s.True(proto.Equal(&planetv1.LandmassHolder{AccountId: ada, CountryId: "fr", Tiles: 2}, holders[0]))
	s.True(proto.Equal(&planetv1.LandmassHolder{AccountId: bob, CountryId: "de", Tiles: 1}, holders[1]))
}
