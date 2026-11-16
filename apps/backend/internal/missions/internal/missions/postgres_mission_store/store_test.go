package postgres_mission_store_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/postgres_mission_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	missions.StoreContractSuite

	db *cppg.Postgres
}

func (s *testSuite) SetupSuite() {
	s.db = cppgtest.StartServer(s.T()).OpenSchema(s.T(), "missions", migrations.FS)
	store := postgres_mission_store.New(s.db)
	s.NewStore = func() missions.Store { return store }
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.StoreContractSuite.SetupTest()
}

func (s *testSuite) TestABriefKeptWithTheChargeItPaidReadsAsItsOrders() {
	day := days.TestingDayAt(time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC))
	account, mission := missions.AccountID{15: 9}, missions.MissionID{15: 1}
	_, err := s.db.On(s.T().Context()).ExecContext(s.T().Context(), `
		INSERT INTO mission_briefs (day, account_id, missions) VALUES ($1, $2, $3::jsonb)
	`, day.String(), uuid.UUID(account), `[{"id": "`+mission.String()+`", "kind": "take_for",
		"reward": {"charge": "shields", "amount": 20}, "payload": {"flag": "fr", "goal": 120}}]`)
	s.Require().NoError(err)

	brief, err := s.NewStore().Brief(s.T().Context(), day, account)

	s.Require().NoError(err)
	s.Equal([]missions.MissionSnapshot{{ID: mission, Order: missions.TakeFor{Flag: "fr", Goal: 120}}}, brief.Snapshot())
}
