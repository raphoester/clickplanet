package postgres_recruitment_store_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals/postgres_recruitment_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	arrivals.RecruitmentStoreContractSuite

	db *cppg.Postgres
}

func (s *testSuite) SetupSuite() {
	s.db = cppgtest.StartServer(s.T()).OpenSchema(s.T(), "auth", migrations.FS)
	store := postgres_recruitment_store.New(s.db)
	s.NewStore = func() arrivals.Recruitments { return store }
	s.Kept = func(arrivals.Recruitments) []arrivals.RecruitmentSnapshot { return s.kept() }
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.RecruitmentStoreContractSuite.SetupTest()
}

func (s *testSuite) kept() []arrivals.RecruitmentSnapshot {
	rows, err := s.db.On(s.T().Context()).QueryContext(s.T().Context(),
		`SELECT arrival_id, recruiter_id FROM recruitments ORDER BY arrival_id`)
	s.Require().NoError(err)
	defer func() { _ = rows.Close() }()

	var kept []arrivals.RecruitmentSnapshot
	for rows.Next() {
		var (
			arrival   uuid.UUID
			recruiter uuid.NullUUID
		)
		s.Require().NoError(rows.Scan(&arrival, &recruiter))
		kept = append(kept, arrivals.RecruitmentSnapshot{
			Arrival: arrivals.ArrivalID(arrival), Recruiter: arrivals.AccountID(recruiter.UUID),
		})
	}
	s.Require().NoError(rows.Err())
	return kept
}

func (s *testSuite) TestAnAnonymizedRecruitmentIsStoredWithNoRecruiter() {
	recruitment, err := arrivals.RecruitmentSnapshot{Arrival: arrivals.ArrivalID{15: 1}}.Restore()
	s.Require().NoError(err)
	s.Require().NoError(postgres_recruitment_store.New(s.db).Save(s.T().Context(), recruitment))

	var recruiter uuid.NullUUID
	s.Require().NoError(s.db.On(s.T().Context()).QueryRowContext(s.T().Context(), `SELECT recruiter_id FROM recruitments`).Scan(&recruiter))
	s.False(recruiter.Valid)
}
