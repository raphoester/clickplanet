package postgres_cast_store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts/postgres_cast_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db    *cppg.Postgres
	store *postgres_cast_store.Store
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "clips", migrations.FS)
	s.store = postgres_cast_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

var madeAt = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

func (s *testSuite) account(last string) casts.AccountID {
	id, err := casts.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a" + last)
	s.Require().NoError(err)
	return id
}

func (s *testSuite) cast(id casts.CastID, at time.Time) casts.CastSnapshot {
	ada, bob := s.account("01"), s.account("02")
	return casts.CastSnapshot{
		ID: id, MadeAt: at, Window: casts.Window{Since: at.Add(-12 * time.Hour), Until: at},
		Fighters: []casts.FighterSnapshot{{Number: 1, Account: bob, Name: "Bob"}, {Number: 2, Account: ada, Name: "Ада"}},
	}
}

func (s *testSuite) TestACastReadsBackAsItWasKept() {
	kept := s.cast(casts.CastID{1}, madeAt)
	s.Require().NoError(s.store.Keep(s.T().Context(), kept.Restore()))

	read, err := s.store.Cast(s.T().Context(), casts.CastID{1})
	s.Require().NoError(err)
	s.Equal(kept, read.TakeSnapshot())
}

func (s *testSuite) TestNoCastIsErrNoCast() {
	_, err := s.store.Cast(s.T().Context(), casts.CastID{1})
	s.ErrorIs(err, casts.ErrNoCast)
}

func (s *testSuite) TestTheCastsMadeBeforeATimeAreFoundAndForgotten() {
	s.Require().NoError(s.store.Keep(s.T().Context(), s.cast(casts.CastID{1}, madeAt.Add(-4*24*time.Hour)).Restore()))
	s.Require().NoError(s.store.Keep(s.T().Context(), s.cast(casts.CastID{2}, madeAt).Restore()))

	old, err := s.store.MadeBefore(s.T().Context(), madeAt.Add(-time.Hour))
	s.Require().NoError(err)
	s.Equal([]casts.CastID{{1}}, old)

	s.Require().NoError(s.store.Forget(s.T().Context(), casts.CastID{1}))
	_, err = s.store.Cast(s.T().Context(), casts.CastID{1})
	s.ErrorIs(err, casts.ErrNoCast)
}

func (s *testSuite) TestADeletedAccountLeavesItsNumberNamingNoOne() {
	s.Require().NoError(s.store.Keep(s.T().Context(), s.cast(casts.CastID{1}, madeAt).Restore()))

	s.Require().NoError(s.store.ForgetAccount(s.T().Context(), s.account("02")))

	read, err := s.store.Cast(s.T().Context(), casts.CastID{1})
	s.Require().NoError(err)
	s.Equal([]casts.FighterSnapshot{{Number: 1}, {Number: 2, Account: s.account("01"), Name: "Ада"}}, read.TakeSnapshot().Fighters)
}
