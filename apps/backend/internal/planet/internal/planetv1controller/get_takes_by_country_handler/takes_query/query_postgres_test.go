package takes_query_test

import (
	"cmp"
	"context"
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
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_takes_by_country_handler/takes_query"
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

var at = time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)

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

func (s *testSuite) takes(account string) *planetv1.GetTakesByCountryResponse {
	id, err := ledger.AccountIDOf(account)
	s.Require().NoError(err)
	answer, err := takes_query.NewPostgresQuery(s.db).TakesByCountry(s.T().Context(), id)
	s.Require().NoError(err)
	return answer
}

func everyKind() []ledger.Event {
	return []ledger.Event{
		ledger.Taking{Tile: 1, Scope: "203.0.113.7", Account: ada, Country: "fr", Previous: "de", At: at},
		ledger.Taking{Tile: 2, Scope: "203.0.113.7", Account: ada, Country: "fr", At: at},
		ledger.Spreading{Tile: 3, Scope: "203.0.113.7", Account: ada, Country: "fr", At: at, Impacts: []clicks.Impact{
			{Tile: 3, Owner: "es", Outcome: clicks.Taken},
			{Tile: 4, Owner: "de", Outcome: clicks.Taken},
			{Tile: 5, Owner: "it", Outcome: clicks.Shielded, Shields: 2},
			{Tile: 6, Outcome: clicks.Taken},
		}},
		ledger.Enclosing{Tile: 7, Scope: "203.0.113.7", Account: ada, Country: "it", At: at, Impacts: []clicks.Impact{
			{Tile: 7, Owner: "fr", Outcome: clicks.Taken},
			{Tile: 8, Owner: "fr", Outcome: clicks.Taken},
		}},
		ledger.Bombing{Scope: "203.0.113.7", Account: ada, At: at, Blast: clicks.Blast{
			Tile: 9, CountryID: "fr", Cleared: []uint32{9, 10}, Owners: []string{"de", "de"},
		}},
		ledger.Striking{Tile: 11, Scope: "203.0.113.7", Account: ada, Country: "fr", Owner: "de", Shields: 1, At: at},
		ledger.Shielding{Tile: 12, Scope: "203.0.113.7", Account: ada, Country: "fr", Shields: 3, At: at},
		ledger.Fortifying{
			Tile: 12, Landmass: 4, Scope: "203.0.113.7", Account: ada, Country: "fr", At: at,
			Raised: []clicks.TileShields{{Tile: 12, Shields: 4}},
		},
		ledger.Taking{Tile: 13, Scope: "198.51.100.4", Account: bob, Country: "de", Previous: "fr", At: at},
		ledger.Taking{Tile: 14, Scope: "198.51.100.4", Country: "fr", Previous: "de", At: at},
	}
}

func (s *testSuite) TestAnAccountTookTilesForEachFlagAndFromTheFlagsThatHeldThemMostFirst() {
	s.record(everyKind()...)

	s.True(proto.Equal(&planetv1.GetTakesByCountryResponse{
		TakenFor:  []*planetv1.CountryTakes{{CountryId: "fr", Tiles: 5}, {CountryId: "it", Tiles: 2}},
		TakenFrom: []*planetv1.CountryTakes{{CountryId: "de", Tiles: 2}, {CountryId: "fr", Tiles: 2}, {CountryId: "es", Tiles: 1}},
	}, s.takes(ada)), "%v", s.takes(ada))
}

func (s *testSuite) TestTheQueryCountsExactlyTheTakesTheLedgerReplaysForTheAccount() {
	events := everyKind()
	s.record(events...)

	takenFor, takenFrom := map[string]uint64{}, map[string]uint64{}
	for _, event := range events {
		event.Replay(func(taking ledger.Taking) {
			if taking.Account != ada || taking.Cleared() {
				return
			}
			takenFor[taking.Country]++
			if taking.Previous != "" && taking.Previous != taking.Country {
				takenFrom[taking.Previous]++
			}
		})
	}

	s.True(proto.Equal(&planetv1.GetTakesByCountryResponse{TakenFor: ranked(takenFor), TakenFrom: ranked(takenFrom)}, s.takes(ada)))
}

func (s *testSuite) TestAnAccountThatTookNothingTookFromNobody() {
	s.record(everyKind()[7:]...)

	answer := s.takes(ada)

	s.Empty(answer.GetTakenFor())
	s.Empty(answer.GetTakenFrom())
}

func (s *testSuite) TestADeletedAccountsTakesAreNobodysAnyMore() {
	s.record(everyKind()...)
	id, err := ledger.AccountIDOf(ada)
	s.Require().NoError(err)

	s.Require().NoError(s.store.AnonymizeTakes(s.T().Context(), id))

	s.Empty(s.takes(ada).GetTakenFor())
}

func (s *testSuite) TestAStoreFailureIsAnError() {
	ctx, cancel := context.WithCancel(s.T().Context())
	cancel()
	id, err := ledger.AccountIDOf(ada)
	s.Require().NoError(err)

	_, err = takes_query.NewPostgresQuery(s.db).TakesByCountry(ctx, id)

	s.Error(err)
}

func ranked(tiles map[string]uint64) []*planetv1.CountryTakes {
	countries := make([]*planetv1.CountryTakes, 0, len(tiles))
	for country, n := range tiles {
		countries = append(countries, &planetv1.CountryTakes{CountryId: country, Tiles: n})
	}
	slices.SortFunc(countries, func(a, b *planetv1.CountryTakes) int {
		return cmp.Or(cmp.Compare(b.GetTiles(), a.GetTiles()), cmp.Compare(a.GetCountryId(), b.GetCountryId()))
	})
	return countries
}
