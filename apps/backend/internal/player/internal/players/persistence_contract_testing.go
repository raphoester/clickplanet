//go:build testing

package players

import (
	"time"

	"github.com/stretchr/testify/suite"
)

// PersistenceContractSuite is the behaviour every Persistence shares. Embed it and set NewPersistence.
type PersistenceContractSuite struct {
	suite.Suite

	// NewPersistence answers an empty persistence.
	NewPersistence func() Persistence

	persistence Persistence
}

func (s *PersistenceContractSuite) SetupTest() {
	s.persistence = s.NewPersistence()
}

var contractAt = time.Date(2026, 9, 17, 12, 30, 0, 123_456_000, time.UTC)

func contractProfile(account byte, name Name) Profile {
	return Profile{Account: AccountID{15: account}, Name: name, UpdatedAt: contractAt}
}

func contractStats(account byte, tiles uint64, day Day) Stats {
	return Stats{Account: AccountID{15: account}, TilesTaken: tiles, StreakCurrent: 2, StreakBest: 5, StreakLastDay: day}
}

func (s *PersistenceContractSuite) save(changes Changes) {
	s.Require().NoError(s.persistence.Save(s.T().Context(), changes))
}

func (s *PersistenceContractSuite) loaded() Snapshot {
	snapshot, err := s.persistence.Load(s.T().Context())
	s.Require().NoError(err)
	return snapshot
}

func (s *PersistenceContractSuite) TestAnEmptyPersistenceLoadsNothing() {
	snapshot := s.loaded()

	s.Empty(snapshot.Profiles)
	s.Empty(snapshot.Stats)
}

func (s *PersistenceContractSuite) TestWhatIsSavedLoadsBack() {
	profiles := []Profile{contractProfile(1, "Émile 🌍"), contractProfile(2, "b")}
	stats := []Stats{contractStats(1, 12, DayOf(contractAt)), contractStats(3, 1, DayOf(contractAt).Following())}

	s.save(Changes{Profiles: profiles, Stats: stats})

	snapshot := s.loaded()
	s.ElementsMatch(profiles, snapshot.Profiles)
	s.ElementsMatch(stats, snapshot.Stats)
}

func (s *PersistenceContractSuite) TestASavedRowReplacesTheOneBefore() {
	s.save(Changes{Profiles: []Profile{contractProfile(1, "before")}, Stats: []Stats{contractStats(1, 1, DayOf(contractAt))}})

	renamed := contractProfile(1, "after")
	renamed.UpdatedAt = contractAt.Add(time.Hour)
	later := contractStats(1, 9, DayOf(contractAt).Following())
	s.save(Changes{Profiles: []Profile{renamed}, Stats: []Stats{later}})

	snapshot := s.loaded()
	s.Equal([]Profile{renamed}, snapshot.Profiles)
	s.Equal([]Stats{later}, snapshot.Stats)
}

func (s *PersistenceContractSuite) TestADeletedRowIsGoneAndTheOthersStay() {
	s.save(Changes{
		Profiles: []Profile{contractProfile(1, "gone"), contractProfile(2, "kept")},
		Stats:    []Stats{contractStats(1, 1, DayOf(contractAt)), contractStats(2, 2, DayOf(contractAt))},
	})

	s.save(Changes{DeletedProfiles: []AccountID{{15: 1}}, DeletedStats: []AccountID{{15: 1}}})

	snapshot := s.loaded()
	s.Equal([]Profile{contractProfile(2, "kept")}, snapshot.Profiles)
	s.Equal([]Stats{contractStats(2, 2, DayOf(contractAt))}, snapshot.Stats)
}

func (s *PersistenceContractSuite) TestDeletingARowNeverSavedIsNotAnError() {
	s.save(Changes{DeletedProfiles: []AccountID{{15: 9}}, DeletedStats: []AccountID{{15: 9}}})

	s.Empty(s.loaded().Profiles)
}

func (s *PersistenceContractSuite) TestAProfileAndStatsAreKeptApart() {
	s.save(Changes{Profiles: []Profile{contractProfile(1, "named")}, Stats: []Stats{contractStats(1, 3, DayOf(contractAt))}})

	s.save(Changes{DeletedStats: []AccountID{{15: 1}}})

	snapshot := s.loaded()
	s.Equal([]Profile{contractProfile(1, "named")}, snapshot.Profiles)
	s.Empty(snapshot.Stats)
}
