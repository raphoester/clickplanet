//go:build testing

package bonuses

import (
	"errors"
	"sync"
	"sync/atomic"

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
	s.Require().NoError(s.store.Change(s.T().Context(), holder, func(Held) (Held, error) {
		return kept, nil
	}))
}

func (s *StoreContractSuite) seen(holder Holder) Held {
	var seen Held
	s.Require().NoError(s.store.Change(s.T().Context(), holder, func(held Held) (Held, error) {
		seen = held
		return held, nil
	}))
	return seen
}

func (s *StoreContractSuite) read(holder Holder) Held {
	held, err := s.store.Held(s.T().Context(), holder)
	s.Require().NoError(err)
	return held
}

func (s *StoreContractSuite) TestAHolderNeverChangedHoldsNothing() {
	s.Equal(Held{}, s.read(s.Holder(1)))
	s.Equal(Held{}, s.seen(s.Holder(1)))
}

func (s *StoreContractSuite) TestEveryChargeIsKeptAndReadBack() {
	full := Held{Refill: true, Bomb: true, Enclosures: 3, SpreadClicks: 8, Shields: 30}
	s.keep(s.Holder(1), full)
	s.keep(s.Holder(2), Held{Shields: 1})

	s.Equal(full, s.read(s.Holder(1)))
	s.Equal(full, s.seen(s.Holder(1)), "a change starts from what was kept")
	s.Equal(Held{Shields: 1}, s.read(s.Holder(2)), "each holder has its own hand")
}

func (s *StoreContractSuite) TestAChangeReplacesWhatWasKept() {
	s.keep(s.Holder(1), Held{Bomb: true, Enclosures: 1})

	s.keep(s.Holder(1), Held{Enclosures: 1})

	s.Equal(Held{Enclosures: 1}, s.read(s.Holder(1)))
}

func (s *StoreContractSuite) TestAHandSpentToNothingHoldsNothing() {
	s.keep(s.Holder(1), Held{Bomb: true})
	s.keep(s.Holder(2), Held{Bomb: true})

	s.keep(s.Holder(1), Held{})

	s.Equal(Held{}, s.read(s.Holder(1)))
	s.Equal(Held{}, s.seen(s.Holder(1)))
	s.Equal(Held{Bomb: true}, s.read(s.Holder(2)))
}

func (s *StoreContractSuite) TestARefusedChangeKeepsNothingAndSaysWhy() {
	s.keep(s.Holder(1), Held{Bomb: true})
	refused := errors.New("refused")

	err := s.store.Change(s.T().Context(), s.Holder(1), func(Held) (Held, error) {
		return Held{}, refused
	})

	s.Require().ErrorIs(err, refused)
	s.Equal(Held{Bomb: true}, s.read(s.Holder(1)))
}

func (s *StoreContractSuite) TestChangesToOneHolderNeverOverlap() {
	const changes = 20

	var wg sync.WaitGroup
	for range changes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.NoError(s.store.Change(s.T().Context(), s.Holder(1), func(held Held) (Held, error) {
				held.Shields++
				return held, nil
			}))
		}()
	}
	wg.Wait()

	s.Equal(changes, s.read(s.Holder(1)).Shields, "each change saw the one before it")
}

func (s *StoreContractSuite) TestTabsRacingForTheLastBombDropOne() {
	const tabs = 10
	s.keep(s.Holder(1), Held{Bomb: true})

	var (
		wg      sync.WaitGroup
		dropped atomic.Int32
	)
	for range tabs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := s.store.Change(s.T().Context(), s.Holder(1), func(held Held) (Held, error) {
				spent, ok := held.AfterBomb()
				if !ok {
					return held, ErrNotHeld
				}
				return spent, nil
			})
			if err == nil {
				dropped.Add(1)
				return
			}
			s.ErrorIs(err, ErrNotHeld)
		}()
	}
	wg.Wait()

	s.Equal(int32(1), dropped.Load())
	s.Equal(Held{}, s.read(s.Holder(1)))
}
