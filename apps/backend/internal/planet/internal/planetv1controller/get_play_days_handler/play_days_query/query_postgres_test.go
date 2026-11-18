package play_days_query_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_play_days_handler/play_days_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
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
	cid = "9c3e1d2b-4a5f-4e6d-8b7c-0a1b2c3d4e5f"
)

// 23:30 UTC on 2026-10-08: half an hour before the next UTC day.
var lateEvening = time.Date(2026, 10, 8, 23, 30, 0, 0, time.UTC)

func onDay(n int) time.Time { return lateEvening.AddDate(0, 0, n) }

func (s *testSuite) SetupSuite() {
	s.db = server.CreateDatabase(s.T()).OpenSchema(s.T(), "planet", migrations.FS)
	s.store = postgres_ledger_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.Require().NoError(eras.Reseed(s.T().Context(), s.db))
}

func (s *testSuite) record(events ...ledger.Event) {
	s.recordMarked(inmemory_ledger_storage.Marks{}, events...)
}

func (s *testSuite) recordMarked(marks inmemory_ledger_storage.Marks, events ...ledger.Event) {
	stored := make([]inmemory_ledger_storage.Stored, 0, len(events))
	for i, event := range events {
		entry, err := event.Entry()
		s.Require().NoError(err)
		entry.Stage = ledger.StageOf(event)
		stored = append(stored, inmemory_ledger_storage.Stored{Position: ledger.Position(i), Entry: entry}) //nolint:gosec // a few events.
	}
	s.Require().NoError(s.store.Save(s.T().Context(), inmemory_ledger_storage.Changes{Entries: slices.Values(stored), Marks: marks}))
}

func (s *testSuite) id(account string) ledger.AccountID {
	id, err := ledger.AccountIDOf(account)
	s.Require().NoError(err)
	return id
}

func (s *testSuite) playDays(accounts ...string) *planetv1.GetPlayDaysResponse {
	ids := make([]ledger.AccountID, 0, len(accounts))
	for _, account := range accounts {
		ids = append(ids, s.id(account))
	}
	answer, err := play_days_query.NewPostgresQuery(s.db).PlayDays(s.T().Context(), ids)
	s.Require().NoError(err)
	return answer
}

// Every kind of event, by ada on day 0 to 8, one kind a day, two viruses' on day 10 and 11, and a mine's on day 12.
func everyKind() []ledger.Event {
	return []ledger.Event{
		ledger.Taking{Tile: 1, Scope: "203.0.113.7", Account: ada, Country: "fr", Previous: "de", At: onDay(0)},
		ledger.Spreading{Tile: 3, Scope: "203.0.113.7", Account: ada, Country: "fr", At: onDay(1), Impacts: []clicks.Impact{
			{Tile: 3, Owner: "es", Outcome: clicks.Taken},
			{Tile: 4, Outcome: clicks.Taken},
		}},
		ledger.Spreading{Tile: 5, Scope: "203.0.113.7", Account: ada, Country: "fr", At: onDay(2), Impacts: []clicks.Impact{
			{Tile: 5, Owner: "it", Outcome: clicks.Shielded, Shields: 2},
		}},
		ledger.Enclosing{Tile: 7, Scope: "203.0.113.7", Account: ada, Country: "it", At: onDay(3), Impacts: []clicks.Impact{
			{Tile: 7, Owner: "fr", Outcome: clicks.Taken},
		}},
		ledger.Bombing{Scope: "203.0.113.7", Account: ada, At: onDay(4), Blast: clicks.Blast{
			Tile: 9, CountryID: "fr", Cleared: []uint32{9, 10}, Owners: []string{"de", "de"},
		}},
		ledger.Striking{Tile: 11, Scope: "203.0.113.7", Account: ada, Country: "fr", Owner: "de", Shields: 1, At: onDay(5)},
		ledger.Shielding{Tile: 12, Scope: "203.0.113.7", Account: ada, Country: "fr", Shields: 3, At: onDay(6)},
		ledger.Fortifying{
			Tile: 12, Landmass: 4, Scope: "203.0.113.7", Account: ada, Country: "fr", At: onDay(7),
			Raised: []clicks.TileShields{{Tile: 12, Shields: 4}},
		},
		ledger.KnockingOut{Tile: 13, Scope: "203.0.113.7", Account: ada, Country: "fr", Out: "de", At: onDay(8), Cleared: []uint32{14}},
		ledger.Taking{Tile: 15, Scope: "198.51.100.4", Account: bob, Country: "de", Previous: "fr", At: onDay(9)},
		ledger.Taking{Tile: 16, Scope: "198.51.100.4", Country: "fr", Previous: "de", At: onDay(9)},
		ledger.Infecting{Tile: 17, Scope: "203.0.113.7", Account: ada, Country: "fr", At: onDay(10), Impacts: []clicks.Impact{
			{Tile: 17, Owner: "de", Outcome: clicks.Taken},
		}},
		ledger.Infecting{Tile: 17, Scope: "203.0.113.7", Account: ada, Country: "fr", At: onDay(11), Impacts: []clicks.Impact{
			{Tile: 18, Owner: "de", Outcome: clicks.Shielded, Shields: 1},
		}},
		ledger.Mining{Tile: 19, Scope: "203.0.113.7", Account: ada, Country: "fr", At: onDay(12), Impacts: []clicks.Impact{
			{Tile: 19, Owner: "de", Outcome: clicks.Taken},
		}},
	}
}

func (s *testSuite) TestEachAccountsDaysAreTheDaysOfTheEventsThatTookATile() {
	s.record(everyKind()...)

	s.True(proto.Equal(&planetv1.GetPlayDaysResponse{Accounts: []*planetv1.AccountPlayDays{
		{AccountId: ada, Days: []string{"2026-10-08", "2026-10-09", "2026-10-11", "2026-10-18", "2026-10-20"}},
		{AccountId: bob, Days: []string{"2026-10-17"}},
	}}, s.playDays(ada, bob)), "%v", s.playDays(ada, bob))
}

func (s *testSuite) TestTheDaysAreTheDaysOfTheTakesTheRelayTells() {
	events := everyKind()
	s.record(events...)

	days := map[string]*cpcolls.Set[string]{}
	for _, event := range events {
		event.Replay(func(taking ledger.Taking) {
			if taking.Account == "" || taking.Cleared() {
				return
			}
			if days[taking.Account] == nil {
				days[taking.Account] = cpcolls.NewSet[string]()
			}
			days[taking.Account].Add(taking.At.UTC().Format(time.DateOnly))
		})
	}

	want := &planetv1.GetPlayDaysResponse{}
	for _, account := range []string{ada, bob} {
		want.Accounts = append(want.Accounts, &planetv1.AccountPlayDays{AccountId: account, Days: sortedKeys(days[account])})
	}
	s.True(proto.Equal(want, s.playDays(ada, bob)), "%v", s.playDays(ada, bob))
}

func sortedKeys(set *cpcolls.Set[string]) []string {
	keys := make([]string, 0, set.Len())
	set.ForEach(func(key string) { keys = append(keys, key) })
	slices.Sort(keys)
	return keys
}

func (s *testSuite) TestADayIsAUTCDayWhateverTheZoneOfTheTake() {
	paris := time.FixedZone("CEST", 2*60*60)
	s.record(
		ledger.Taking{Tile: 1, Account: ada, Country: "fr", At: time.Date(2026, 10, 9, 1, 0, 0, 0, paris)},
		ledger.Taking{Tile: 2, Account: ada, Country: "fr", At: lateEvening},
		ledger.Taking{Tile: 3, Account: ada, Country: "fr", At: lateEvening.Add(time.Hour)},
	)

	s.True(proto.Equal(&planetv1.GetPlayDaysResponse{Accounts: []*planetv1.AccountPlayDays{
		{AccountId: ada, Days: []string{"2026-10-08", "2026-10-09"}},
	}}, s.playDays(ada)), "%v", s.playDays(ada))
}

func (s *testSuite) TestOnlyTheAccountsAskedAreAnsweredAndOneWithNoTakeIsLeftOut() {
	s.record(everyKind()...)

	answer := s.playDays(bob, cid)

	s.Require().Len(answer.GetAccounts(), 1)
	s.Equal(bob, answer.GetAccounts()[0].GetAccountId())
}

func (s *testSuite) TestARevertedTakeStillCounts() {
	s.recordMarked(inmemory_ledger_storage.Marks{
		Head:      1,
		Forgotten: map[ledger.Caller]ledger.Position{{Account: ada}: 1},
	}, ledger.Taking{Tile: 1, Scope: "203.0.113.7", Account: ada, Country: "fr", At: lateEvening})

	s.Len(s.playDays(ada).GetAccounts(), 1)
}

func (s *testSuite) TestADeletedAccountsTakesAreNobodysAnyMore() {
	s.record(everyKind()...)

	s.Require().NoError(s.store.AnonymizeTakes(s.T().Context(), s.id(ada)))

	s.Empty(s.playDays(ada).GetAccounts())
}
