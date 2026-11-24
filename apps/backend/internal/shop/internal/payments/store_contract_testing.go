//go:build testing

package payments

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
	paidAt = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	buyer  = orders.AccountID{15: 1}
)

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

func (s *StoreContractSuite) paid(id PaymentID, intent IntentID) Payment {
	payment, _, err := Pay(Charged{
		ID: id, Order: TestOrder(buyer, plans.Week, paidAt.Add(-time.Minute)), Intent: intent, Amount: 400,
		Currency: "eur", At: paidAt,
	})
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), payment))
	return payment
}

func (s *StoreContractSuite) TestAPaymentIsReadAsItWasKept() {
	payment := s.paid("cs_1", "pi_1")

	kept, err := s.store.Payment(s.T().Context(), "cs_1")
	s.Require().NoError(err)
	s.Equal(payment.TakeSnapshot(), kept.TakeSnapshot())
}

func (s *StoreContractSuite) TestAPaymentIsFoundByItsIntent() {
	payment := s.paid("in_1", "pi_1")
	s.paid("cs_2", "pi_2")

	kept, err := s.store.OfIntent(s.T().Context(), "pi_1")
	s.Require().NoError(err)
	s.Equal(payment.TakeSnapshot(), kept.TakeSnapshot())
}

func (s *StoreContractSuite) TestAPaymentNeverKeptIsUnknown() {
	_, err := s.store.Payment(s.T().Context(), "cs_1")
	s.Require().ErrorIs(err, ErrUnknownPayment)
	_, err = s.store.OfIntent(s.T().Context(), "pi_1")
	s.Require().ErrorIs(err, ErrUnknownPayment)
}

func (s *StoreContractSuite) TestAPaymentTakenBackKeepsItsState() {
	payment := s.paid("cs_1", "pi_1")
	taken, _, _ := payment.TakenBack(paidAt.Add(time.Hour))
	s.Require().NoError(s.store.Save(s.T().Context(), taken))

	kept, err := s.store.Payment(s.T().Context(), "cs_1")
	s.Require().NoError(err)
	s.Equal(taken.TakeSnapshot(), kept.TakeSnapshot())
}

func (s *StoreContractSuite) TestAFlagsPaymentKeepsItsFlag() {
	target, err := orders.ParseTarget("", "r-0000000001")
	s.Require().NoError(err)
	payment, _, err := Pay(Charged{
		ID: "in_1", Order: orders.Snapshot{Buyer: buyer, Plan: plans.Year, Target: target, AgreedAt: paidAt},
		Intent: "pi_1", Amount: 6000, Currency: "eur", At: paidAt,
	})
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), payment))

	kept, err := s.store.Payment(s.T().Context(), "in_1")
	s.Require().NoError(err)
	s.Equal(payment.TakeSnapshot(), kept.TakeSnapshot())
}
