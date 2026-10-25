//go:build testing

package bonuses

import (
	"errors"
	"sync"

	"github.com/stretchr/testify/suite"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store
	Holder   func(n int) Holder

	store Store
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

func (s *StoreContractSuite) keep(holder Holder, kept Held) {
	s.Require().NoError(s.store.Change(s.T().Context(), holder, func(Held) (Held, error) { return kept, nil }))
}

func (s *StoreContractSuite) held(holder Holder) Held {
	held, err := s.store.Held(s.T().Context(), holder)
	s.Require().NoError(err)
	return held
}

func (s *StoreContractSuite) TestEveryChargeIsKeptAndEachHolderHasItsOwn() {
	full := Held{Refill: true, Bomb: true, Enclosures: 3, SpreadClicks: 8, Shields: 30, LastFill: "season-0:finale:fill"}
	s.keep(s.Holder(1), full)
	s.keep(s.Holder(2), Held{Shields: 1})

	s.Equal(full, s.held(s.Holder(1)))
	s.Equal(Held{Shields: 1}, s.held(s.Holder(2)))
	s.Equal(Held{}, s.held(s.Holder(3)), "a holder never changed holds nothing")
}

func (s *StoreContractSuite) TestAHandSpentToNothingHoldsNothing() {
	s.keep(s.Holder(1), Held{Bomb: true})

	s.keep(s.Holder(1), Held{})

	s.Equal(Held{}, s.held(s.Holder(1)))
}

func (s *StoreContractSuite) TestAHandSpentToNothingKeepsItsLastFill() {
	s.keep(s.Holder(1), Held{Bomb: true, LastFill: "season-0:finale:fill"})

	s.keep(s.Holder(1), Held{LastFill: "season-0:finale:fill"})

	s.Equal(Held{LastFill: "season-0:finale:fill"}, s.held(s.Holder(1)))
}

func (s *StoreContractSuite) TestARefusedChangeKeepsNothingAndSaysWhy() {
	s.keep(s.Holder(1), Held{Bomb: true})
	refused := errors.New("refused")

	err := s.store.Change(s.T().Context(), s.Holder(1), func(Held) (Held, error) { return Held{}, refused })

	s.Require().ErrorIs(err, refused)
	s.Equal(Held{Bomb: true}, s.held(s.Holder(1)))
}

func (s *StoreContractSuite) TestChangesToOneHolderNeverOverlap() {
	const changes = 20

	var wg sync.WaitGroup
	for range changes {
		wg.Go(func() {
			s.NoError(s.store.Change(s.T().Context(), s.Holder(1), func(held Held) (Held, error) {
				held.Shields++
				return held, nil
			}))
		})
	}
	wg.Wait()

	s.Equal(changes, s.held(s.Holder(1)).Shields, "each change saw the one before it, so two tabs never spend one charge")
}
