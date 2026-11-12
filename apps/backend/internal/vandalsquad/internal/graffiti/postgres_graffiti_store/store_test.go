package postgres_graffiti_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti/postgres_graffiti_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/migrations"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	graffiti.StorageContractSuite

	db *cppg.Postgres
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "vandalsquad", migrations.FS)
	store := postgres_graffiti_store.New(s.db)
	s.NewStorage = func() graffiti.Storage { return store }
	s.Kept = func(graffiti.Storage) []graffiti.Snapshot { return s.kept() }
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.StorageContractSuite.SetupTest()
}

func (s *testSuite) kept() []graffiti.Snapshot {
	rows, err := s.db.On(s.T().Context()).QueryContext(s.T().Context(), `
		SELECT id, account, flag, kind, tiles, flagged_at, seen_at, link FROM graffiti ORDER BY flagged_at, id
	`)
	s.Require().NoError(err)
	defer func() { _ = rows.Close() }()

	var kept []graffiti.Snapshot
	for rows.Next() {
		snapshot, err := postgres_graffiti_store.Scan(rows)
		s.Require().NoError(err)
		kept = append(kept, snapshot)
	}
	s.Require().NoError(rows.Err())
	return kept
}
