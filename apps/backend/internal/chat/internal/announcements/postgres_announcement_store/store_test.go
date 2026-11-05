package postgres_announcement_store_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/postgres_announcement_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	announcements.StorageContractSuite

	db *cppg.Postgres
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "chat", migrations.FS)
	store := postgres_announcement_store.New(s.db)
	s.NewStorage = func() announcements.Storage { return store }
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.StorageContractSuite.SetupTest()
}

func (s *testSuite) TestAKindTheTypeDoesNotHoldIsRefused() {
	meteor := announcements.NewAnnouncement(announcements.AnnouncementID{15: 1}, announcements.Kind("meteor"),
		time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC), json.RawMessage(`{}`))

	_, err := s.NewStorage().AppendOnce(s.T().Context(), meteor)
	s.Error(err)
}
