//go:build testing

package notices

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
	contractAt = time.Date(2026, 11, 15, 12, 30, 0, 123456000, time.UTC)

	ada   = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"
	grace = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a11"
	alan  = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a12"
)

func (s *StoreContractSuite) keep(notice Notice) bool {
	kept, err := s.store.Keep(s.T().Context(), notice)
	s.Require().NoError(err)
	return kept
}

func (s *StoreContractSuite) noticed(account string) (Notice, bool) {
	notice, noticed, err := s.store.Noticed(s.T().Context(), account)
	s.Require().NoError(err)
	return notice, noticed
}

func (s *StoreContractSuite) same(want Notice, account string) {
	got, noticed := s.noticed(account)
	s.Require().True(noticed)
	s.Equal(want.Account, got.Account)
	s.Equal(want.Alongside, got.Alongside)
	s.True(want.At.Equal(got.At), "at: want %s, got %s", want.At, got.At)
	s.True(want.AcknowledgedAt.Equal(got.AcknowledgedAt), "acknowledged at: want %s, got %s", want.AcknowledgedAt, got.AcknowledgedAt)
}

func (s *StoreContractSuite) TestANoticeIsKeptAndReadBack() {
	s.True(s.keep(Notice{Account: ada, Alongside: grace, At: contractAt}))

	s.same(Notice{Account: ada, Alongside: grace, At: contractAt}, ada)
	_, noticed := s.noticed(grace)
	s.False(noticed, "only the account noticed is in")
}

func (s *StoreContractSuite) TestTheFirstNoticeIsKept() {
	s.True(s.keep(Notice{Account: ada, Alongside: grace, At: contractAt}))

	s.False(s.keep(Notice{Account: ada, Alongside: alan, At: contractAt.Add(time.Hour)}))

	s.same(Notice{Account: ada, Alongside: grace, At: contractAt}, ada)
}

func (s *StoreContractSuite) TestANoticeIsPendingUntilAcknowledged() {
	s.keep(Notice{Account: ada, Alongside: grace, At: contractAt})
	notice, _ := s.noticed(ada)
	s.True(notice.Pending())

	s.Require().NoError(s.store.Acknowledge(s.T().Context(), ada, contractAt.Add(time.Minute)))

	s.same(Notice{Account: ada, Alongside: grace, At: contractAt, AcknowledgedAt: contractAt.Add(time.Minute)}, ada)
}

func (s *StoreContractSuite) TestTheFirstAcknowledgementIsKept() {
	s.keep(Notice{Account: ada, Alongside: grace, At: contractAt})
	s.Require().NoError(s.store.Acknowledge(s.T().Context(), ada, contractAt.Add(time.Minute)))

	s.Require().NoError(s.store.Acknowledge(s.T().Context(), ada, contractAt.Add(time.Hour)))

	s.same(Notice{Account: ada, Alongside: grace, At: contractAt, AcknowledgedAt: contractAt.Add(time.Minute)}, ada)
}

func (s *StoreContractSuite) TestAnAccountNeverNoticedAcknowledgesNothing() {
	s.Require().NoError(s.store.Acknowledge(s.T().Context(), ada, contractAt))

	_, noticed := s.noticed(ada)
	s.False(noticed)
}
