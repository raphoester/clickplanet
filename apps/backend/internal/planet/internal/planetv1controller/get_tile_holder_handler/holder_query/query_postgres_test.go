package holder_query_test

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas/postgres_arena_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/postgres_ledger_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_tile_holder_handler/holder_query"
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
}

const (
	ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	bob = "5a0c2f1e-7d3b-4c8a-9e6f-1b2d3c4e5f60"
)

var at = time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)

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

func (s *testSuite) flush(events int, head ledger.Position, forgotten map[ledger.Caller]ledger.Position) {
	s.Require().NoError(s.store.Save(s.T().Context(), inmemory_ledger_storage.Changes{
		From:    ledger.Position(events), //nolint:gosec // a few events.
		Entries: slices.Values([]inmemory_ledger_storage.Stored{}),
		Marks:   inmemory_ledger_storage.Marks{Head: head, Forgotten: forgotten},
	}))
}

func (s *testSuite) forget(caller ledger.Caller, events int) {
	s.flush(events, 0, map[ledger.Caller]ledger.Position{caller: ledger.Position(events)}) //nolint:gosec // a few events.
}

func (s *testSuite) holder(tile uint32) *planetv1.TileHolder {
	answer, err := holder_query.NewPostgresQuery(s.db, postgres_arena_store.New(s.db), cptime.NewFixedClock(at.Add(24*time.Hour))).
		TileHolder(s.T().Context(), tile)
	s.Require().NoError(err)
	return answer.GetHolder()
}

func minute(n int) time.Time {
	return at.Add(time.Duration(n) * time.Minute)
}

func everyKind() []ledger.Event {
	return []ledger.Event{
		ledger.Taking{Tile: 1, Scope: "203.0.113.7", Account: ada, Country: "fr", Previous: "de", At: minute(0)},
		ledger.Taking{Tile: 1, Scope: "198.51.100.4", Account: bob, Country: "de", Previous: "fr", At: minute(1)},
		ledger.Taking{Tile: 2, Scope: "203.0.113.7", Account: ada, Country: "fr", At: minute(2)},
		ledger.Spreading{Tile: 2, Scope: "198.51.100.4", Account: bob, Country: "de", At: minute(3), Impacts: []clicks.Impact{
			{Tile: 2, Owner: "fr", Outcome: clicks.Taken},
			{Tile: 3, Owner: "es", Outcome: clicks.Taken},
			{Tile: 4, Owner: "it", Outcome: clicks.Shielded, Shields: 2},
		}},
		ledger.Taking{Tile: 4, Scope: "203.0.113.7", Account: ada, Country: "it", At: minute(4)},
		ledger.Enclosing{Tile: 5, Scope: "203.0.113.7", Account: ada, Country: "it", At: minute(5), Impacts: []clicks.Impact{
			{Tile: 5, Owner: "fr", Outcome: clicks.Taken},
			{Tile: 6, Owner: "fr", Outcome: clicks.Taken},
		}},
		ledger.Taking{Tile: 7, Scope: "198.51.100.4", Account: bob, Country: "de", At: minute(6)},
		ledger.Bombing{Scope: "203.0.113.7", Account: ada, At: minute(7), Blast: clicks.Blast{
			Tile: 7, CountryID: "fr", Cleared: []uint32{7, 6}, Owners: []string{"de", "it"},
		}},
		ledger.Taking{Tile: 8, Scope: "203.0.113.7", Account: ada, Country: "fr", At: minute(8)},
		ledger.Striking{Tile: 8, Scope: "198.51.100.4", Account: bob, Country: "de", Owner: "fr", Shields: 1, At: minute(9)},
		ledger.Shielding{Tile: 8, Scope: "198.51.100.4", Account: bob, Country: "fr", Shields: 3, At: minute(10)},
		ledger.Fortifying{
			Tile: 8, Landmass: 4, Scope: "198.51.100.4", Account: bob, Country: "fr", At: minute(11),
			Raised: []clicks.TileShields{{Tile: 8, Shields: 4}},
		},
		ledger.Taking{Tile: 9, Scope: "198.51.100.4", Country: "fr", Previous: "de", At: minute(12)},
		ledger.Taking{Tile: 10, Scope: "203.0.113.7", Account: ada, At: minute(13)},
	}
}

func (s *testSuite) TestTheLastTakeOfATileHoldsIt() {
	s.record(everyKind()...)

	s.True(proto.Equal(&planetv1.TileHolder{
		AccountId: bob, CountryId: "de", TakenAtUnixMs: minute(1).UnixMilli(),
	}, s.holder(1)), "%v", s.holder(1))
}

func (s *testSuite) TestATileASpreadOrAnEncloseTookIsHeldByItsPlayer() {
	s.record(everyKind()...)

	s.True(proto.Equal(&planetv1.TileHolder{AccountId: bob, CountryId: "de", TakenAtUnixMs: minute(3).UnixMilli()}, s.holder(3)))
	s.True(proto.Equal(&planetv1.TileHolder{AccountId: ada, CountryId: "it", TakenAtUnixMs: minute(5).UnixMilli()}, s.holder(5)))
}

func (s *testSuite) TestATileABombClearedLastIsHeldByNobody() {
	s.record(everyKind()...)

	s.Nil(s.holder(7))
	s.Nil(s.holder(6))
}

func (s *testSuite) TestAStrikeAShieldOrAFortifyLeavesTheTileWithItsTaker() {
	s.record(everyKind()...)

	s.True(proto.Equal(&planetv1.TileHolder{AccountId: ada, CountryId: "fr", TakenAtUnixMs: minute(8).UnixMilli()}, s.holder(8)))
}

func (s *testSuite) TestATakeThatNamesNoAccountIsHeldWithNoAccount() {
	s.record(everyKind()...)

	s.True(proto.Equal(&planetv1.TileHolder{CountryId: "fr", TakenAtUnixMs: minute(12).UnixMilli()}, s.holder(9)))
}

func (s *testSuite) TestADeletedAccountsTakeStillHoldsTheTile() {
	s.record(everyKind()...)
	account, err := ledger.AccountIDOf(bob)
	s.Require().NoError(err)
	s.Require().NoError(s.store.AnonymizeTakes(s.T().Context(), account))

	s.True(proto.Equal(&planetv1.TileHolder{CountryId: "de", TakenAtUnixMs: minute(1).UnixMilli()}, s.holder(1)))
	s.True(proto.Equal(&planetv1.TileHolder{CountryId: "de", TakenAtUnixMs: minute(3).UnixMilli()}, s.holder(3)))
}

func (s *testSuite) TestATileARevertGaveBackIsHeldByTheTakeBeforeTheRevertedOnes() {
	events := everyKind()
	s.record(events...)

	s.forget(ledger.Caller{Account: bob}, len(events))

	s.True(proto.Equal(&planetv1.TileHolder{AccountId: ada, CountryId: "fr", TakenAtUnixMs: minute(0).UnixMilli()}, s.holder(1)))
	s.True(proto.Equal(&planetv1.TileHolder{AccountId: ada, CountryId: "fr", TakenAtUnixMs: minute(2).UnixMilli()}, s.holder(2)),
		"the tile a forgotten spread took")
	s.Nil(s.holder(3), "a tile only a forgotten spread took")
}

func (s *testSuite) TestATakeARevertOfItsScopeForgotHoldsNothing() {
	events := everyKind()
	s.record(events...)

	s.forget(ledger.Caller{Scope: "198.51.100.4"}, len(events))

	s.True(proto.Equal(&planetv1.TileHolder{AccountId: ada, CountryId: "fr", TakenAtUnixMs: minute(0).UnixMilli()}, s.holder(1)))
	s.Nil(s.holder(9), "a take with no account, from the forgotten scope")
}

func (s *testSuite) TestAForgottenTakeStaysForgottenOnceItsMarkLeftWithTheHeadAndItsAccountIsDeleted() {
	events := everyKind()
	s.record(events...)
	s.forget(ledger.Caller{Account: bob}, len(events))
	s.flush(len(events), ledger.Position(len(events)), map[ledger.Caller]ledger.Position{})
	account, err := ledger.AccountIDOf(bob)
	s.Require().NoError(err)

	s.Require().NoError(s.store.AnonymizeTakes(s.T().Context(), account))

	s.True(proto.Equal(&planetv1.TileHolder{AccountId: ada, CountryId: "fr", TakenAtUnixMs: minute(0).UnixMilli()}, s.holder(1)))
}

func (s *testSuite) TestATileNobodyEverTookIsHeldByNobody() {
	s.record(everyKind()...)

	s.Nil(s.holder(11))
}

func (s *testSuite) TestTheHolderIsTheLastTakeTheLedgerReplaysForEachTile() {
	events := everyKind()
	s.record(events...)

	last := map[uint32]ledger.Taking{}
	for _, event := range events {
		event.Replay(func(taking ledger.Taking) { last[taking.Tile] = taking })
	}

	for tile := uint32(1); tile <= 11; tile++ {
		taking, changed := last[tile]
		if !changed || taking.Cleared() {
			s.Nil(s.holder(tile), "tile %d", tile)
			continue
		}
		s.True(proto.Equal(&planetv1.TileHolder{
			AccountId: taking.Account, CountryId: taking.Country, TakenAtUnixMs: taking.At.UnixMilli(),
		}, s.holder(tile)), "tile %d: %v", tile, s.holder(tile))
	}
}

func (s *testSuite) arena(id arenas.ID, openedAt, closedAt time.Time) {
	ctx := s.T().Context()
	kept := postgres_arena_store.New(s.db)
	s.Require().NoError(kept.Open(ctx, arenas.Opening{ID: id, OpenedAt: openedAt, EndsAt: openedAt.Add(time.Hour)}))
	if !closedAt.IsZero() {
		s.Require().NoError(kept.Close(ctx, arenas.Closing{ID: id, ClosedAt: closedAt}))
	}
}

func (s *testSuite) TestWhileAnArenaPlaysOnlyItsTakesHoldATile() {
	s.arena("battle", at.Add(time.Hour), time.Time{})
	s.record(
		ledger.Taking{Tile: 7, Scope: "203.0.113.7", Account: ada, Country: "fr", At: at},
		ledger.Taking{Tile: 8, Scope: "203.0.113.7", Account: ada, Country: "fr", At: at},
		ledger.Taking{Tile: 8, Scope: "198.51.100.4", Account: bob, Country: "fr", At: at.Add(2 * time.Hour)},
	)

	s.Nil(s.holder(7), "a capital nobody took in the arena names nobody, whoever held it on the world's map")
	s.Equal(bob, s.holder(8).GetAccountId())
}

func (s *testSuite) TestAfterAnArenaItsTakesHoldNoTile() {
	closed := at.Add(2*time.Hour + 700*time.Millisecond)
	s.arena("battle", at.Add(time.Hour), closed)
	s.record(
		ledger.Taking{Tile: 7, Scope: "203.0.113.7", Account: ada, Country: "fr", At: at},
		ledger.Taking{Tile: 7, Scope: "198.51.100.4", Account: bob, Country: "fr", At: at.Add(90 * time.Minute)},
		ledger.Taking{Tile: 8, Scope: "198.51.100.4", Account: bob, Country: "fr", At: closed.Truncate(time.Second)},
	)

	s.Equal(ada, s.holder(7).GetAccountId(), "the world's take, not the arena's")
	s.Equal(bob, s.holder(8).GetAccountId(), "a world take in the second the arena closed")
}
