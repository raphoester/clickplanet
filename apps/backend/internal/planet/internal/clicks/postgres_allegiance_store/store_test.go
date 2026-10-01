package postgres_allegiance_store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/postgres_allegiance_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite
	store *postgres_allegiance_store.Store
	db    *cppg.Postgres
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "planet", migrations.FS)
	s.store = postgres_allegiance_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

var (
	epoch = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	ada   = clicks.AccountAllegianceKey("ada")
	home  = clicks.ScopeAllegianceKey("2001:db8::/64")
)

func (s *testSuite) TestATallyReadsBackAsItWasSaved() {
	tally := clicks.Allegiance{}.With("pl", epoch).With("pl", epoch.Add(time.Hour)).With("ad", epoch.Add(time.Hour))

	s.Require().NoError(s.store.SaveAllegiances(s.T().Context(), map[clicks.AllegianceKey]clicks.Allegiance{ada: tally}))

	tallies, err := s.store.Allegiances(s.T().Context(), ada, home)
	s.Require().NoError(err)
	s.Equal(map[clicks.AllegianceKey]clicks.Allegiance{ada: tally}, tallies, "a key with no row is absent")
}

func (s *testSuite) TestASaveReplacesTheTally() {
	first := clicks.Allegiance{}.With("pl", epoch)
	second := first.With("ad", epoch.Add(time.Hour))

	s.Require().NoError(s.store.SaveAllegiances(s.T().Context(), map[clicks.AllegianceKey]clicks.Allegiance{ada: first}))
	s.Require().NoError(s.store.SaveAllegiances(s.T().Context(), map[clicks.AllegianceKey]clicks.Allegiance{ada: second, home: first}))

	tallies, err := s.store.Allegiances(s.T().Context(), ada, home)
	s.Require().NoError(err)
	s.Equal(map[clicks.AllegianceKey]clicks.Allegiance{ada: second, home: first}, tallies)
}

func (s *testSuite) TestTalliesWithNoTakeSinceTheCutoffAreDeleted() {
	s.Require().NoError(s.store.SaveAllegiances(s.T().Context(), map[clicks.AllegianceKey]clicks.Allegiance{
		ada:  clicks.Allegiance{}.With("pl", epoch),
		home: clicks.Allegiance{}.With("pl", epoch.Add(time.Hour)),
	}))

	deleted, err := s.store.DeleteAllegiancesBefore(s.T().Context(), epoch.Add(time.Minute))
	s.Require().NoError(err)
	s.Equal(int64(1), deleted)

	tallies, err := s.store.Allegiances(s.T().Context(), ada, home)
	s.Require().NoError(err)
	s.Equal([]clicks.AllegianceKey{home}, keysOf(tallies))
}

func keysOf(tallies map[clicks.AllegianceKey]clicks.Allegiance) []clicks.AllegianceKey {
	keys := make([]clicks.AllegianceKey, 0, len(tallies))
	for key := range tallies {
		keys = append(keys, key)
	}
	return keys
}
