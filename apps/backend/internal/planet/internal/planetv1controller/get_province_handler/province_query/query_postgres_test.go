package province_query_test

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/postgres_ledger_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_province_handler/province_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db      *cppg.Postgres
	store   *postgres_ledger_store.Store
	playing clicks.Stage
	cut     string
}

// stage is the map that plays, on its board.
type stage clicks.Stage

func (s stage) Stage() clicks.Stage { return clicks.Stage(s) }

// cut is the provinces file in play.
type cut string

func (c cut) Provinces() *clicks.Provinces {
	borders := clicks.BordersOf(4, []uint32{1, 2}, []uint32{3, 4})
	return clicks.ProvincesOf(string(c), borders, []uint32{1}, []uint32{2}, []uint32{3, 4})
}

const (
	ada      = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	bob      = "5a0c2f1e-7d3b-4c8a-9e6f-1b2d3c4e5f60"
	firstCut = "provinces-286f4c3e.bin"
)

var at = time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)

func (s *testSuite) SetupSuite() {
	s.db = server.CreateDatabase(s.T()).OpenSchema(s.T(), "planet", migrations.FS)
	s.store = postgres_ledger_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.Require().NoError(eras.Reseed(s.T().Context(), s.db))
	s.playing = clicks.WorldStage
	s.cut = firstCut
}

func (s *testSuite) record(events ...ledger.Event) {
	stored := make([]inmemory_ledger_storage.Stored, 0, len(events))
	for i, event := range events {
		entry, err := event.Entry()
		s.Require().NoError(err)
		entry.Stage = ledger.StageOf(event)
		stored = append(stored, inmemory_ledger_storage.Stored{Position: ledger.Position(i), Entry: entry}) //nolint:gosec // a few events.
	}
	s.Require().NoError(s.store.Save(s.T().Context(), inmemory_ledger_storage.Changes{Entries: slices.Values(stored)}))
}

func (s *testSuite) province(province clicks.ProvinceID) *planetv1.GetProvinceResponse {
	answer, err := province_query.NewPostgresQuery(s.db, stage(s.playing), cut(s.cut)).Province(s.T().Context(), province)
	s.Require().NoError(err)
	return answer
}

func fortify(province clicks.ProvinceID, account, flag string, minute int) ledger.Fortifying {
	return ledger.Fortifying{
		Tile: 1, Province: province, Provinces: firstCut, Ground: "it", Tiles: 3, Scope: "203.0.113.7",
		Account: account, Country: flag, At: at.Add(time.Duration(minute) * time.Minute),
	}
}

func (s *testSuite) TestTheLastFortifyOfTheProvinceIsTheLatestOfItsOwn() {
	s.record(fortify(1, ada, "fr", 0), fortify(1, bob, "de", 5), fortify(2, ada, "fr", 9))

	s.True(proto.Equal(&planetv1.ProvinceFortification{
		AccountId: bob, CountryId: "de", FortifiedAtUnixMs: at.Add(5 * time.Minute).UnixMilli(),
	}, s.province(1).GetLastFortification()))
}

func (s *testSuite) TestAProvinceNobodyFortifiedHasNoLastFortify() {
	s.record(fortify(2, ada, "fr", 0))

	s.Nil(s.province(1).GetLastFortification())
}

func (s *testSuite) TestAFortifyMadeBeforeProvincesIsNoProvincesFortify() {
	s.record(ledger.Fortifying{Tile: 1, Landmass: 1, Ground: "it", Tiles: 3, Account: ada, Country: "fr", At: at})

	s.Nil(s.province(1).GetLastFortification(), "landmass 1 is no province")
}

func (s *testSuite) TestAFortifyOfAnotherCutIsAnotherPlace() {
	s.record(fortify(1, ada, "fr", 0))
	s.cut = "provinces-0a0b0c0d.bin"

	s.Nil(s.province(1).GetLastFortification(), "province 1 of another cut is another place")
}

func (s *testSuite) TestAFortifyWhoseAccountWasDeletedNamesNobody() {
	s.record(fortify(1, "", "fr", 0))

	s.Empty(s.province(1).GetLastFortification().GetAccountId())
	s.Equal("fr", s.province(1).GetLastFortification().GetCountryId())
}

func (s *testSuite) TestTheLastFortifyIsOfTheMapThatPlaysOnItsBoard() {
	battle := clicks.MapID("battle").On(clicks.TestBoard)
	s.record(
		fortify(1, ada, "fr", 0),
		ledger.Staged{On: battle, Event: fortify(1, bob, "br", 20)},
		ledger.Staged{On: clicks.World.On(eras.Earth2.ID), Event: fortify(1, bob, "de", 30)},
	)
	s.playing = battle

	s.Equal(bob, s.province(1).GetLastFortification().GetAccountId(), "while the arena plays, its own fortify")

	s.playing = clicks.WorldStage

	s.Equal("fr", s.province(1).GetLastFortification().GetCountryId(),
		"once it closed, the world's on its board: province 1 of another board is another place")
}
