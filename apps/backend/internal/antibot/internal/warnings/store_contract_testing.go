//go:build testing

package warnings

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

var (
	contractAt = time.Date(2026, 11, 10, 12, 30, 0, 123456000, time.UTC)

	ada   = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"
	grace = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a11"
	alan  = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a12"
)

func (s *StoreContractSuite) keep(warning Warning) bool {
	kept, err := s.store.Keep(s.T().Context(), warning)
	s.Require().NoError(err)
	return kept
}

func (s *StoreContractSuite) warned(accounts ...string) map[string]Warning {
	warned, err := s.store.Warned(s.T().Context(), accounts)
	s.Require().NoError(err)
	return warned
}

func (s *StoreContractSuite) same(want, got Warning) {
	s.Equal(want.Account, got.Account)
	s.True(want.At.Equal(got.At), "at: want %s, got %s", want.At, got.At)
	s.True(want.AcknowledgedAt.Equal(got.AcknowledgedAt), "acknowledged at: want %s, got %s", want.AcknowledgedAt, got.AcknowledgedAt)
}

func (s *StoreContractSuite) TestAnAccountNeverWarnedIsNotIn() {
	s.Empty(s.warned(ada))
}

func (s *StoreContractSuite) TestAWarningIsKeptAndReadBack() {
	s.True(s.keep(Warning{Account: ada, At: contractAt}))

	warned := s.warned(ada, grace)

	s.Len(warned, 1)
	s.same(Warning{Account: ada, At: contractAt}, warned[ada])
}

func (s *StoreContractSuite) TestTheFirstWarningIsKept() {
	s.True(s.keep(Warning{Account: ada, At: contractAt}))

	s.False(s.keep(Warning{Account: ada, At: contractAt.Add(time.Hour)}))

	s.same(Warning{Account: ada, At: contractAt}, s.warned(ada)[ada])
}

func (s *StoreContractSuite) TestNoAccountsReadNothing() {
	s.keep(Warning{Account: ada, At: contractAt})

	s.Empty(s.warned())
}

func (s *StoreContractSuite) TestSinceAnswersTheWarningsFromATime() {
	s.keep(Warning{Account: ada, At: contractAt.Add(-time.Hour)})
	s.keep(Warning{Account: grace, At: contractAt})
	s.keep(Warning{Account: alan, At: contractAt.Add(time.Hour)})

	since, err := s.store.Since(s.T().Context(), contractAt)

	s.Require().NoError(err)
	s.Len(since, 2)
	accounts := map[string]Warning{}
	for _, warning := range since {
		accounts[warning.Account] = warning
	}
	s.same(Warning{Account: grace, At: contractAt}, accounts[grace])
	s.same(Warning{Account: alan, At: contractAt.Add(time.Hour)}, accounts[alan])
}

func (s *StoreContractSuite) TestAWarningIsPendingUntilAcknowledged() {
	s.keep(Warning{Account: ada, At: contractAt})
	s.True(s.warned(ada)[ada].Pending())

	s.Require().NoError(s.store.Acknowledge(s.T().Context(), ada, contractAt.Add(time.Minute)))

	s.same(Warning{Account: ada, At: contractAt, AcknowledgedAt: contractAt.Add(time.Minute)}, s.warned(ada)[ada])
}

func (s *StoreContractSuite) TestTheFirstAcknowledgementIsKept() {
	s.keep(Warning{Account: ada, At: contractAt})
	s.Require().NoError(s.store.Acknowledge(s.T().Context(), ada, contractAt.Add(time.Minute)))

	s.Require().NoError(s.store.Acknowledge(s.T().Context(), ada, contractAt.Add(time.Hour)))

	s.same(Warning{Account: ada, At: contractAt, AcknowledgedAt: contractAt.Add(time.Minute)}, s.warned(ada)[ada])
}

func (s *StoreContractSuite) TestAnAccountNeverWarnedAcknowledgesNothing() {
	s.Require().NoError(s.store.Acknowledge(s.T().Context(), ada, contractAt))

	s.Empty(s.warned(ada))
}
