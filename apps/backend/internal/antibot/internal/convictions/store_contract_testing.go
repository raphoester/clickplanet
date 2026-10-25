//go:build testing

package convictions

import (
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store
	Account  func(n int) string

	store Store
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

var contractAt = time.Date(2026, 10, 31, 21, 15, 0, 123456000, time.UTC)

func (s *StoreContractSuite) keep(conviction Conviction) {
	s.Require().NoError(s.store.Keep(s.T().Context(), conviction))
}

func (s *StoreContractSuite) between(since, until time.Time) []Conviction {
	between, err := s.store.Between(s.T().Context(), since, until)
	s.Require().NoError(err)
	return between
}

func (s *StoreContractSuite) TestAConvictionIsReadBackWhole() {
	accomplice := Conviction{
		At: contractAt, Scope: "2001:db8:1::/64", Account: s.Account(2), Alongside: s.Account(1),
		Flags: 3, Offence: 2, BannedUntil: contractAt.Add(7 * 24 * time.Hour),
		Readings: []detect.Reading{
			{Watchdog: "retaker", Level: "clear"},
			{Watchdog: "metronome", Level: "suspect", Evidence: "cadence clicks=120 spread=40ms", At: contractAt.Add(-time.Minute)},
			{Watchdog: "rotator", Level: "suspect", Evidence: "turns accounts=2", At: contractAt},
		},
		Clicks: 412, ActiveFor: 42*time.Minute + 1500*time.Microsecond, LongestGap: 2500 * time.Millisecond,
		TopCountry: "dz", TopCountryClicks: 400,
	}
	s.keep(accomplice)

	s.Equal([]Conviction{accomplice}, s.between(contractAt, contractAt.Add(time.Second)))
}

func (s *StoreContractSuite) TestAConvictionWithNoAccountIsTheScopes() {
	scope := Conviction{
		At: contractAt, Scope: "203.0.113.7", Flags: 1, Offence: 1, BannedUntil: contractAt.Add(24 * time.Hour),
		Readings: []detect.Reading{{Watchdog: "sequencer", Level: "certain", Evidence: "walk steps=200", At: contractAt}},
	}
	s.keep(scope)

	s.Equal([]Conviction{scope}, s.between(contractAt.Add(-time.Hour), contractAt.Add(time.Hour)))
}

func (s *StoreContractSuite) TestAWindowHoldsItsSinceAndNotItsUntilOldestFirst() {
	at := func(minutes int) Conviction {
		return Conviction{
			At: contractAt.Add(time.Duration(minutes) * time.Minute), Scope: "203.0.113.7", Account: s.Account(minutes),
			Flags: 1, Offence: 1, BannedUntil: contractAt.Add(24 * time.Hour), Readings: []detect.Reading{},
		}
	}
	s.keep(at(3))
	s.keep(at(0))
	s.keep(at(2))
	s.keep(at(1))

	s.Equal([]Conviction{at(1), at(2)}, s.between(at(1).At, at(3).At))
}

func (s *StoreContractSuite) TestNothingKeptIsNoConviction() {
	s.Empty(s.between(contractAt.Add(-time.Hour), contractAt))
}
