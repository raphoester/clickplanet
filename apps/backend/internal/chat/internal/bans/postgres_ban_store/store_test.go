package postgres_ban_store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans/postgres_ban_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite
	db    *cppg.Postgres
	store *postgres_ban_store.Store
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "chat", migrations.FS)
	s.store = postgres_ban_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

var noon = time.Date(2026, 9, 16, 12, 0, 0, 123_456_000, time.UTC)

func ban(tag string, reason string) bans.Ban {
	return bans.Ban{AuthorTag: tag, BannedAt: noon, Reason: reason}
}

func (s *testSuite) all() []bans.Ban {
	all, err := s.store.All(context.Background())
	s.Require().NoError(err)
	return all
}

func (s *testSuite) TestAnEmptyTableHasNoBans() {
	s.Empty(s.all())
}

func (s *testSuite) TestABanComesBackAsItWentIn() {
	s.Require().NoError(s.store.Upsert(context.Background(), ban("a1b2c3", "spam")))

	s.Equal([]bans.Ban{ban("a1b2c3", "spam")}, s.all())
}

func (s *testSuite) TestBanningTheSameTagRewritesItRatherThanFailing() {
	s.Require().NoError(s.store.Upsert(context.Background(), ban("a1b2c3", "spam")))
	s.Require().NoError(s.store.Upsert(context.Background(), ban("a1b2c3", "still spam")))

	s.Equal([]bans.Ban{ban("a1b2c3", "still spam")}, s.all())
}

func (s *testSuite) TestDeletingLiftsOneBanAndLeavesTheRest() {
	s.Require().NoError(s.store.Upsert(context.Background(), ban("a1b2c3", "spam")))
	s.Require().NoError(s.store.Upsert(context.Background(), ban("d4e5f6", "spam")))

	s.Require().NoError(s.store.Delete(context.Background(), "a1b2c3"))

	s.Equal([]bans.Ban{ban("d4e5f6", "spam")}, s.all())
}

func (s *testSuite) TestDeletingWhatIsNotThereIsNotAnError() {
	s.Require().NoError(s.store.Delete(context.Background(), "a1b2c3"))
}
