package postgres_removal_store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals/postgres_removal_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db    *cppg.Postgres
	store *postgres_removal_store.Store
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "chat", migrations.FS)
	s.store = postgres_removal_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

var start = time.Date(2026, 10, 18, 12, 0, 0, 123_456_000, time.UTC)

func (s *testSuite) kept() map[string]time.Time {
	rows, err := s.db.On(s.T().Context()).QueryContext(s.T().Context(), `SELECT message_id, removed_at FROM removals`)
	s.Require().NoError(err)
	defer func() { _ = rows.Close() }()

	kept := map[string]time.Time{}
	for rows.Next() {
		var (
			id string
			at time.Time
		)
		s.Require().NoError(rows.Scan(&id, &at))
		kept[id] = at.UTC()
	}
	s.Require().NoError(rows.Err())
	return kept
}

func (s *testSuite) TestARemovalIsKeptOncePerMessage() {
	s.Require().NoError(s.store.Save(s.T().Context(), removals.Snapshot{MessageID: "insult", RemovedAt: start}))
	s.Require().NoError(s.store.Save(s.T().Context(), removals.Snapshot{MessageID: "insult", RemovedAt: start.Add(time.Minute)}))

	s.Equal(map[string]time.Time{"insult": start.Add(time.Minute)}, s.kept(), "removing twice keeps the latest")
}

func (s *testSuite) TestThePruneDeletesTheRemovalsBeforeTheCutoff() {
	s.Require().NoError(s.store.Save(s.T().Context(), removals.Snapshot{MessageID: "old", RemovedAt: start}))
	s.Require().NoError(s.store.Save(s.T().Context(), removals.Snapshot{MessageID: "new", RemovedAt: start.Add(time.Hour)}))

	deleted, err := s.store.DeleteBefore(s.T().Context(), start.Add(time.Minute))
	s.Require().NoError(err)

	s.Equal(int64(1), deleted)
	s.Equal(map[string]time.Time{"new": start.Add(time.Hour)}, s.kept())
}
