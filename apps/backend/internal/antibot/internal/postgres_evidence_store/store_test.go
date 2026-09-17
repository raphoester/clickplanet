package postgres_evidence_store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/evidence"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/postgres_evidence_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite
	db    *cppg.Postgres
	store *postgres_evidence_store.Store
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "antibot", migrations.FS)
	s.store = postgres_evidence_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

// savedAt is whole microseconds, the precision postgres keeps.
var savedAt = time.Date(2026, 9, 15, 12, 0, 0, 654321000, time.UTC)

func (s *testSuite) TestAnEmptyTableLoadsAnEmptySnapshot() {
	snapshot, err := s.store.Load(s.T().Context())

	s.Require().NoError(err)
	s.Empty(snapshot.Sections)
	s.True(snapshot.SavedAt.IsZero())
}

func (s *testSuite) TestSaveThenLoad() {
	s.Require().NoError(s.store.Save(s.T().Context(), evidence.Snapshot{
		SavedAt:  savedAt,
		Sections: map[string][]byte{"jury": {1, 2, 3}, "metronome": {}},
	}))

	snapshot, err := s.store.Load(s.T().Context())

	s.Require().NoError(err)
	s.Equal(map[string][]byte{"jury": {1, 2, 3}, "metronome": {}}, snapshot.Sections)
	s.True(snapshot.SavedAt.Equal(savedAt))
}

func (s *testSuite) TestSaveReplacesEverySection() {
	ctx := s.T().Context()
	s.Require().NoError(s.store.Save(ctx, evidence.Snapshot{
		SavedAt:  savedAt,
		Sections: map[string][]byte{"jury": {1}, "retaker": {2}},
	}))

	s.Require().NoError(s.store.Save(ctx, evidence.Snapshot{
		SavedAt:  savedAt.Add(time.Minute),
		Sections: map[string][]byte{"jury": {3}},
	}))

	snapshot, err := s.store.Load(ctx)
	s.Require().NoError(err)
	s.Equal(map[string][]byte{"jury": {3}}, snapshot.Sections, "a section left out of a flush is gone")
	s.True(snapshot.SavedAt.Equal(savedAt.Add(time.Minute)))
}

func (s *testSuite) TestAFailedSaveWritesNothing() {
	ctx := s.T().Context()
	s.Require().NoError(s.store.Save(ctx, evidence.Snapshot{SavedAt: savedAt, Sections: map[string][]byte{"jury": {1}}}))

	s.Require().Error(s.store.Save(ctx, evidence.Snapshot{
		SavedAt:  savedAt.Add(time.Minute),
		Sections: map[string][]byte{"retaker": {2}, "": {3}},
	}))

	snapshot, err := s.store.Load(ctx)
	s.Require().NoError(err)
	s.Equal(map[string][]byte{"jury": {1}}, snapshot.Sections)
}
