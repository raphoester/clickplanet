//go:build testing

package unlocks

import (
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store

	store Store
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

const (
	ada = bonuses.Account("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	bob = bonuses.Account("1c7e5a8f-6e8d-4b47-8b62-4a2a9a1d3b22")
)

var contractStart = time.Date(2026, 11, 16, 12, 0, 0, 123_000_000, time.UTC)

func (s *StoreContractSuite) decide(account bonuses.Account, kinds bonuses.Unlocked, at time.Time) {
	s.Require().NoError(s.store.Change(s.T().Context(), account, func(u Unlocks) Unlocks {
		return u.Decided(DecisionOf(kinds, at))
	}))
}

func (s *StoreContractSuite) unlocked(account bonuses.Account) bonuses.Unlocked {
	kinds, err := s.store.Unlocked(s.T().Context(), account)
	s.Require().NoError(err)
	return kinds
}

func (s *StoreContractSuite) TestAnAccountNeverToldOfHasNoKindUnlocked() {
	s.Empty(s.unlocked(ada))
}

func (s *StoreContractSuite) TestTheKindsDecidedAreKeptForTheirAccountOnly() {
	s.decide(ada, bonuses.Unlocked{bonuses.KindEncloseClicks, bonuses.KindBomb}, contractStart)

	s.ElementsMatch(bonuses.Unlocked{bonuses.KindEncloseClicks, bonuses.KindBomb}, s.unlocked(ada))
	s.Empty(s.unlocked(bob))
}

func (s *StoreContractSuite) TestALaterDecisionReplacesTheKeptOneAndAnOlderOneDoesNot() {
	s.decide(ada, bonuses.Unlocked{bonuses.KindEncloseClicks}, contractStart.Add(time.Hour))
	s.decide(ada, bonuses.Unlocked{bonuses.KindEncloseClicks, bonuses.KindBomb}, contractStart)

	s.Equal(bonuses.Unlocked{bonuses.KindEncloseClicks}, s.unlocked(ada), "the raise came after the reconciliation")

	s.decide(ada, bonuses.Unlocked{}, contractStart.Add(2*time.Hour))

	s.Empty(s.unlocked(ada))
}

func (s *StoreContractSuite) TestTheTimeOfTheKeptDecisionIsReadBack() {
	s.decide(ada, bonuses.Unlocked{bonuses.KindBomb}, contractStart)

	var kept Unlocks
	s.Require().NoError(s.store.Change(s.T().Context(), ada, func(u Unlocks) Unlocks {
		kept = u
		return u
	}))

	s.Equal(Snapshot{Kinds: bonuses.Unlocked{bonuses.KindBomb}, DecidedAt: contractStart}, kept.TakeSnapshot())
}

func (s *StoreContractSuite) TestAForgottenAccountHasNoKindUnlockedAgain() {
	s.decide(ada, bonuses.Unlocked{bonuses.KindBomb}, contractStart)
	s.decide(bob, bonuses.Unlocked{bonuses.KindBomb}, contractStart)

	s.Require().NoError(s.store.Forget(s.T().Context(), ada))
	s.Require().NoError(s.store.Forget(s.T().Context(), ada), "twice is not an error")

	s.Empty(s.unlocked(ada))
	s.NotEmpty(s.unlocked(bob))
}
