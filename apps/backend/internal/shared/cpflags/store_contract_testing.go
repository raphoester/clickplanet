//go:build testing

package cpflags

import (
	"time"

	"github.com/stretchr/testify/suite"
)

// StoreContractSuite is what every Store keeps.
type StoreContractSuite struct {
	suite.Suite
	NewStore func() Store

	store Store
}

var (
	openedAt = time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	closedAt = openedAt.Add(time.Hour)
)

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

func (s *StoreContractSuite) heard(flag string) HeardSnapshot {
	heard, err := s.store.Heard(s.T().Context(), flag)
	s.Require().NoError(err)
	return heard.TakeSnapshot()
}

func (s *StoreContractSuite) change(flag string, change func(Heard) Heard) {
	s.Require().NoError(s.store.Change(s.T().Context(), flag, change))
}

func (s *StoreContractSuite) TestAFlagNoFactNamedWasNeverHeard() {
	s.Equal(HeardSnapshot{}, s.heard("r-0000000000"))
}

func (s *StoreContractSuite) TestAFlagIsReadAsTheLatestFactLeftIt() {
	s.change("r-0000000000", func(heard Heard) Heard { return heard.Opened(openedAt) })
	s.Equal(HeardSnapshot{State: Open, DecidedAt: openedAt}, s.heard("r-0000000000"))

	s.change("r-0000000000", func(heard Heard) Heard { return heard.Closed(closedAt) })
	s.Equal(HeardSnapshot{State: Closed, DecidedAt: closedAt}, s.heard("r-0000000000"))
}

func (s *StoreContractSuite) TestEachFlagIsKeptApart() {
	s.change("r-0000000000", func(heard Heard) Heard { return heard.Opened(openedAt) })
	s.change("r-1111111111", func(heard Heard) Heard { return heard.Closed(closedAt) })

	s.Equal(HeardSnapshot{State: Open, DecidedAt: openedAt}, s.heard("r-0000000000"))
	s.Equal(HeardSnapshot{State: Closed, DecidedAt: closedAt}, s.heard("r-1111111111"))
}

func (s *StoreContractSuite) TestAChangeIsHandedWhatWasKept() {
	s.change("r-0000000000", func(heard Heard) Heard { return heard.Closed(closedAt) })

	var handed HeardSnapshot
	s.change("r-0000000000", func(heard Heard) Heard {
		handed = heard.TakeSnapshot()
		return heard
	})
	s.Equal(HeardSnapshot{State: Closed, DecidedAt: closedAt}, handed)
}
