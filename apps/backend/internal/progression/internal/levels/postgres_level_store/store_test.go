package postgres_level_store_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/postgres_level_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	levels.StoreContractSuite

	db    *cppg.Postgres
	store *postgres_level_store.Store
}

func (s *testSuite) SetupSuite() {
	s.db = cppgtest.StartServer(s.T()).OpenSchema(s.T(), "progression", migrations.FS)
	s.store = postgres_level_store.New(s.db)
	s.NewStore = func() levels.Store { return s.store }
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.StoreContractSuite.SetupTest()
}

var (
	ada = accounts.AccountID{15: 1}
	at  = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
)

func (s *testSuite) reachedAt() time.Time {
	var reachedAt time.Time
	s.Require().NoError(s.db.On(s.T().Context()).QueryRowContext(s.T().Context(),
		`SELECT reached_at FROM levels WHERE account_id = $1`, ada.String()).Scan(&reachedAt))
	return reachedAt.UTC()
}

func (s *testSuite) TestALevelKeepsTheTimeItWasReached() {
	_, err := s.store.Raise(s.T().Context(), ada, 4, at)
	s.Require().NoError(err)

	_, err = s.store.Raise(s.T().Context(), ada, 4, at.Add(time.Hour))
	s.Require().NoError(err)
	s.Require().NoError(s.store.RaiseAll(s.T().Context(), levels.ByAccount{ada: 3}, at.Add(time.Hour)))

	s.Equal(at, s.reachedAt(), "no raise, no new time")

	_, err = s.store.Raise(s.T().Context(), ada, 5, at.Add(2*time.Hour))
	s.Require().NoError(err)
	s.Equal(at.Add(2*time.Hour), s.reachedAt())
}

func (s *testSuite) TestRaisesAtOnceKeepTheHighestAndEachJumpStartsWhereTheOneBeforeEnded() {
	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		failed error
		twice  bool
		jumps  = map[levels.Level]levels.Level{}
	)
	for level := range levels.Level(12) {
		wg.Go(func() {
			previous, err := s.store.Raise(s.T().Context(), ada, level+1, at)
			mu.Lock()
			defer mu.Unlock()
			failed = errors.Join(failed, err)
			if err == nil && previous < level+1 {
				_, read := jumps[previous]
				twice = twice || read
				jumps[previous] = level + 1
			}
		})
	}
	wg.Wait()

	s.Require().NoError(failed)
	s.False(twice, "two raises read the same level")
	kept, err := s.store.Levels(s.T().Context(), []accounts.AccountID{ada})
	s.Require().NoError(err)
	s.Equal(levels.Level(12), kept[ada])
	for from := levels.Level(0); from != 12; from = jumps[from] {
		s.Require().Contains(jumps, from, "a jump ended on %d and none started there", from)
	}
}

func (s *testSuite) TestARaiseOfAPageNeverLowersARaiseMadeAtOnce() {
	var (
		wg               sync.WaitGroup
		raised, raiseAll error
	)
	wg.Go(func() { _, raised = s.store.Raise(s.T().Context(), ada, 9, at) })
	wg.Go(func() { raiseAll = s.store.RaiseAll(s.T().Context(), levels.ByAccount{ada: 5}, at) })
	wg.Wait()

	s.Require().NoError(raised)
	s.Require().NoError(raiseAll)
	kept, err := s.store.Levels(s.T().Context(), []accounts.AccountID{ada})
	s.Require().NoError(err)
	s.Equal(levels.Level(9), kept[ada])
}
