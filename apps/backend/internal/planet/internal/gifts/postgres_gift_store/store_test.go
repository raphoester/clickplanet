package postgres_gift_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/gifts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/gifts/postgres_gift_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	gifts.StorageContractSuite

	db *cppg.Postgres
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "planet", migrations.FS)
	s.NewStorage = func() gifts.Storage { return postgres_gift_store.New(s.db) }
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.StorageContractSuite.SetupTest()
}

const ada bonuses.Holder = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"

func (s *testSuite) TestAGiftOutlivesTheStoreThatGaveIt() {
	s.Require().NoError(postgres_gift_store.New(s.db).Give(s.T().Context(), "finale-0", ada))

	restarted := postgres_gift_store.New(s.db)

	s.Require().ErrorIs(restarted.Give(s.T().Context(), "finale-0", ada), gifts.ErrGiven)
}
