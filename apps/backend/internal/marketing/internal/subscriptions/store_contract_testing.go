//go:build testing

package subscriptions

import (
	"time"

	"github.com/stretchr/testify/suite"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store

	store Store
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

var contractAt = time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)

func (s *StoreContractSuite) subscription(account byte, address Address, state State) *Subscription {
	return &Subscription{Account: AccountID{15: account}, Address: address, State: state, Consent: SeasonEmails, AskedAt: contractAt}
}

func (s *StoreContractSuite) save(subscription *Subscription) {
	s.Require().NoError(s.store.Save(s.T().Context(), subscription))
}

func (s *StoreContractSuite) found(account byte) *Subscription {
	found, err := s.store.Subscription(s.T().Context(), AccountID{15: account})
	s.Require().NoError(err)
	return found
}

func (s *StoreContractSuite) TestAnAccountThatNeverAskedHasNoSubscription() {
	_, err := s.store.Subscription(s.T().Context(), AccountID{15: 1})

	s.ErrorIs(err, ErrNoSubscription)
}

func (s *StoreContractSuite) TestASavedSubscriptionReadsBack() {
	s.save(s.subscription(1, "ada@example.com", StateWaiting))

	s.Equal(s.subscription(1, "ada@example.com", StateWaiting), s.found(1))
}

func (s *StoreContractSuite) TestASecondSaveReplacesTheFirstOne() {
	s.save(s.subscription(1, "ada@example.com", StateWaiting))

	s.save(s.subscription(1, "ada@example.com", StateWaiting).Confirmed())

	s.Equal(StateActive, s.found(1).State)
}

func (s *StoreContractSuite) TestAWithdrawalKeepsTheRowAndItsTime() {
	withdrawn := s.subscription(1, "ada@example.com", StateActive).Withdrawn(contractAt.Add(time.Hour))

	s.save(withdrawn)

	s.Equal(withdrawn, s.found(1))
}

func (s *StoreContractSuite) TestAnOptInAfterAWithdrawalIsTheSameRowAskedAgain() {
	s.save(s.subscription(1, "ada@example.com", StateActive).Withdrawn(contractAt.Add(time.Hour)))

	again := s.subscription(1, "ada@example.com", StateActive)
	again.AskedAt = contractAt.Add(2 * time.Hour)
	s.save(again)

	s.Equal(again, s.found(1))
	s.True(s.found(1).WithdrawnAt.IsZero())
}

func (s *StoreContractSuite) TestTheSubscriptionsToAnAddressAreEveryAccountThatGaveIt() {
	s.save(s.subscription(1, "ada@example.com", StateActive))
	s.save(s.subscription(2, "ada@example.com", StateWaiting))
	s.save(s.subscription(3, "bob@example.com", StateActive))

	found, err := s.store.SubscriptionsTo(s.T().Context(), "ada@example.com")

	s.Require().NoError(err)
	s.ElementsMatch([]*Subscription{
		s.subscription(1, "ada@example.com", StateActive),
		s.subscription(2, "ada@example.com", StateWaiting),
	}, found)
}

func (s *StoreContractSuite) TestNoSubscriptionToAnAddressNobodyGaveIsNotAnError() {
	found, err := s.store.SubscriptionsTo(s.T().Context(), "nobody@example.com")

	s.Require().NoError(err)
	s.Empty(found)
}

func (s *StoreContractSuite) TestADeletedAccountLosesItsRowAndTheOthersKeepTheirs() {
	s.save(s.subscription(1, "ada@example.com", StateActive))
	s.save(s.subscription(2, "bob@example.com", StateActive))

	s.Require().NoError(s.store.Delete(s.T().Context(), AccountID{15: 1}))

	_, err := s.store.Subscription(s.T().Context(), AccountID{15: 1})
	s.ErrorIs(err, ErrNoSubscription)
	s.Equal(StateActive, s.found(2).State)
}

func (s *StoreContractSuite) TestDeletingAnAccountWithNoRowIsNotAnError() {
	s.Require().NoError(s.store.Delete(s.T().Context(), AccountID{15: 9}))
}
