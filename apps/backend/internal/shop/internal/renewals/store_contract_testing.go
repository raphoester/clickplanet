//go:build testing

package renewals

import (
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/plans"
)

// StoreContractSuite is what every Store keeps.
type StoreContractSuite struct {
	suite.Suite
	NewStore func() Store

	store Store
}

var (
	toldAt = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	buyer  = orders.AccountID{15: 1}
)

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

func (s *StoreContractSuite) told(id SubscriptionID, account orders.AccountID, ends time.Time) Renewal {
	renewal, err := TestRenewing(id, account, ends, toldAt).Restore()
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), renewal))
	return renewal
}

func (s *StoreContractSuite) TestARenewalIsReadAsItWasKept() {
	renewal := s.told("sub_1", buyer, toldAt.AddDate(0, 1, 0))

	kept, err := s.store.Renewal(s.T().Context(), "sub_1")
	s.Require().NoError(err)
	s.Equal(renewal.TakeSnapshot(), kept.TakeSnapshot())
}

func (s *StoreContractSuite) TestARenewalNeverKeptIsUnknown() {
	_, err := s.store.Renewal(s.T().Context(), "sub_1")
	s.Require().ErrorIs(err, ErrUnknownRenewal)
}

func (s *StoreContractSuite) TestABuyersRenewalsComeByTheEndOfTheirPeriod() {
	later := s.told("sub_1", buyer, toldAt.AddDate(1, 0, 0))
	sooner := s.told("sub_2", buyer, toldAt.AddDate(0, 1, 0))
	s.told("sub_3", orders.AccountID{15: 2}, toldAt.AddDate(0, 1, 0))

	bought, err := s.store.Bought(s.T().Context(), buyer)
	s.Require().NoError(err)
	s.Require().Len(bought, 2)
	s.Equal(sooner.TakeSnapshot(), bought[0].TakeSnapshot())
	s.Equal(later.TakeSnapshot(), bought[1].TakeSnapshot())
}

func (s *StoreContractSuite) TestARenewalStoppedKeepsItsState() {
	renewal := s.told("sub_1", buyer, toldAt.AddDate(0, 1, 0))
	stopped, _, err := renewal.StoppedBy(buyer, toldAt.Add(time.Hour))
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), stopped))

	kept, err := s.store.Renewal(s.T().Context(), "sub_1")
	s.Require().NoError(err)
	s.Equal(stopped.TakeSnapshot(), kept.TakeSnapshot())
}

// TestRenewing is a month the account bought for a submission, renewing until a time, as Stripe told it then.
func TestRenewing(id SubscriptionID, account orders.AccountID, ends, at time.Time) Snapshot {
	target, err := orders.ParseTarget("00000000-0000-0000-0000-000000000001", "")
	if err != nil {
		panic(err)
	}
	return Snapshot{
		ID: id, Order: orders.Snapshot{Buyer: account, Plan: plans.Month, Target: target, AgreedAt: at.Add(-time.Minute)},
		State: Renewing, PeriodEndsAt: ends, DecidedAt: at,
	}
}
