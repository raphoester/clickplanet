package postgres_ban_store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans/inmemory_ban_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans/postgres_ban_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	inmemory_ban_storage.PersistenceContractSuite

	db    *cppg.Postgres
	store *postgres_ban_store.Store
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "chat", migrations.FS)
	s.store = postgres_ban_store.New(s.db)

	// The contract's own SetupTest calls this, so the purge is what empties it per test.
	s.NewPersistence = func() inmemory_ban_storage.Persistence {
		s.Require().NoError(s.db.Purge(context.Background()))
		return s.store
	}
}

func (s *testSuite) TestTheBanTimeKeepsItsMicroseconds() {
	at := time.Date(2026, 9, 16, 12, 0, 0, 123_456_000, time.UTC)
	s.Require().NoError(s.store.Upsert(context.Background(),
		bans.Ban{AuthorTag: "a1b2c3", BannedAt: at, Reason: "spam"}))

	all, err := s.store.All(context.Background())
	s.Require().NoError(err)
	s.Require().Len(all, 1)
	s.Equal(at, all[0].BannedAt)
}
