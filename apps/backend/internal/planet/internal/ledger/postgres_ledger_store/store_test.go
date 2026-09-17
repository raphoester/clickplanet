package postgres_ledger_store_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/postgres_ledger_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
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

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "planet", migrations.FS)
	s.store = postgres_ledger_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

var start = time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)

func stored(position ledger.Position, tile uint32, scope, country, previous string, at time.Time) inmemory_ledger_storage.Stored {
	return inmemory_ledger_storage.Stored{
		Position: position,
		Taking:   ledger.Taking{Tile: tile, Scope: scope, Country: country, Previous: previous, At: at},
	}
}

func changes(from, head ledger.Position, forgotten map[ledger.Caller]ledger.Position, takes ...inmemory_ledger_storage.Stored) inmemory_ledger_storage.Changes {
	return inmemory_ledger_storage.Changes{
		From:  from,
		Takes: slices.Values(takes),
		Marks: inmemory_ledger_storage.Marks{Head: head, Forgotten: forgotten},
	}
}

func (s *testSuite) load() ([]inmemory_ledger_storage.Stored, inmemory_ledger_storage.Marks) {
	var takes []inmemory_ledger_storage.Stored
	marks, err := s.store.Load(context.Background(), func(take inmemory_ledger_storage.Stored) {
		takes = append(takes, take)
	})
	s.Require().NoError(err)
	return takes, marks
}

func (s *testSuite) TestAnEmptyStoreLoadsNothing() {
	takes, marks := s.load()

	s.Empty(takes)
	s.Equal(inmemory_ledger_storage.Marks{Forgotten: map[ledger.Caller]ledger.Position{}}, marks)
}

func (s *testSuite) TestSaveThenLoadInPositionOrder() {
	ctx := context.Background()
	first := []inmemory_ledger_storage.Stored{
		stored(0, 7, "1.2.3.4", "fr", "de", start),
		stored(1, 257_948, "2001:db8::/64", "ps", "", start.Add(time.Second)),
	}
	second := stored(5, 8, "1.2.3.4", "fr", "", start.Add(time.Minute))

	s.Require().NoError(s.store.Save(ctx, changes(0, 0, map[ledger.Caller]ledger.Position{}, first...)))
	s.Require().NoError(s.store.Save(ctx, changes(5, 0, map[ledger.Caller]ledger.Position{{Scope: "bot"}: 2}, second)))

	takes, marks := s.load()
	s.Equal(slices.Concat(first, []inmemory_ledger_storage.Stored{second}), takes)
	s.Equal(inmemory_ledger_storage.Marks{Forgotten: map[ledger.Caller]ledger.Position{{Scope: "bot"}: 2}}, marks)
}

func (s *testSuite) TestATimeComesBackInUTC() {
	paris := time.FixedZone("CEST", 2*60*60)
	s.Require().NoError(s.store.Save(context.Background(), changes(0, 0, map[ledger.Caller]ledger.Position{}, stored(0, 1, "a", "fr", "", start.In(paris)))))

	takes, _ := s.load()
	s.Require().Len(takes, 1)
	s.Equal(start, takes[0].Taking.At)
}

func (s *testSuite) TestASaveWhoseCommitWasLostIsWrittenAgainWithoutConflict() {
	ctx := context.Background()
	take := stored(3, 1, "a", "fr", "", start)

	s.Require().NoError(s.store.Save(ctx, changes(3, 0, map[ledger.Caller]ledger.Position{}, take)))
	s.Require().NoError(s.store.Save(ctx, changes(3, 0, map[ledger.Caller]ledger.Position{}, take)))

	takes, _ := s.load()
	s.Equal([]inmemory_ledger_storage.Stored{take}, takes)
}

func (s *testSuite) TestTheHeadDeletesOlderTakesAndMarks() {
	ctx := context.Background()
	s.Require().NoError(s.store.Save(ctx, changes(0, 0, map[ledger.Caller]ledger.Position{{Scope: "old"}: 1, {Scope: "new"}: 3},
		stored(0, 1, "old", "fr", "", start),
		stored(1, 2, "new", "fr", "", start),
		stored(2, 3, "new", "fr", "", start),
	)))

	s.Require().NoError(s.store.Save(ctx, changes(3, 2, map[ledger.Caller]ledger.Position{})))

	takes, marks := s.load()
	s.Equal([]inmemory_ledger_storage.Stored{stored(2, 3, "new", "fr", "", start)}, takes)
	s.Equal(inmemory_ledger_storage.Marks{Head: 2, Forgotten: map[ledger.Caller]ledger.Position{{Scope: "new"}: 3}}, marks)
}

func (s *testSuite) TestAMarkOnlyMovesForward() {
	ctx := context.Background()
	s.Require().NoError(s.store.Save(ctx, changes(0, 0, map[ledger.Caller]ledger.Position{{Scope: "bot"}: 5})))
	s.Require().NoError(s.store.Save(ctx, changes(0, 0, map[ledger.Caller]ledger.Position{{Scope: "bot"}: 4})))

	_, marks := s.load()
	s.Equal(map[ledger.Caller]ledger.Position{{Scope: "bot"}: 5}, marks.Forgotten)
}

func (s *testSuite) TestSaveCopiesManyTakes() {
	const count = 100_000
	takes := make([]inmemory_ledger_storage.Stored, count)
	for i := range takes {
		takes[i] = stored(ledger.Position(i), uint32(i), "1.2.3.4", "fr", "", start)
	}

	s.Require().NoError(s.store.Save(context.Background(), changes(0, 0, map[ledger.Caller]ledger.Position{}, takes...)))

	loaded, _ := s.load()
	s.Len(loaded, count)
}

func (s *testSuite) TestAFailedSaveWritesNothing() {
	ctx := context.Background()
	s.Require().NoError(s.store.Save(ctx, changes(0, 0, map[ledger.Caller]ledger.Position{}, stored(0, 1, "a", "fr", "", start))))

	// The second tile overflows its integer column; the first take and the marks must not land either.
	err := s.store.Save(ctx, changes(1, 1, map[ledger.Caller]ledger.Position{{Scope: "bot"}: 2},
		stored(1, 2, "a", "fr", "", start),
		stored(2, 1<<31, "a", "fr", "", start),
	))
	s.Require().Error(err)

	takes, marks := s.load()
	s.Equal([]inmemory_ledger_storage.Stored{stored(0, 1, "a", "fr", "", start)}, takes)
	s.Equal(inmemory_ledger_storage.Marks{Forgotten: map[ledger.Caller]ledger.Position{}}, marks)
}

func (s *testSuite) TestATakesAccountComesBackAndNoneIsNull() {
	ctx := context.Background()
	const guest = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"
	withAccount := stored(0, 1, "1.2.3.4", "fr", "", start)
	withAccount.Taking.Account = guest
	without := stored(1, 2, "1.2.3.4", "fr", "", start)

	s.Require().NoError(s.store.Save(ctx, changes(0, 0, map[ledger.Caller]ledger.Position{}, withAccount, without)))

	var nulls int
	s.Require().NoError(s.db.QueryRowContext(ctx, `SELECT count(*) FROM ledger_takes WHERE account IS NULL`).Scan(&nulls))
	s.Equal(1, nulls)

	takes, _ := s.load()
	s.Equal([]inmemory_ledger_storage.Stored{withAccount, without}, takes)
}

func (s *testSuite) TestAccountMarksAreKeptApartFromScopeMarks() {
	ctx := context.Background()
	const guest = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"
	marks := map[ledger.Caller]ledger.Position{{Scope: "bot"}: 2, {Account: guest}: 4}

	s.Require().NoError(s.store.Save(ctx, changes(0, 0, marks)))
	_, loaded := s.load()
	s.Equal(marks, loaded.Forgotten)

	s.Require().NoError(s.store.Save(ctx, changes(0, 3, map[ledger.Caller]ledger.Position{})))
	_, loaded = s.load()
	s.Equal(map[ledger.Caller]ledger.Position{{Account: guest}: 4}, loaded.Forgotten, "the head drops account marks too")
}
