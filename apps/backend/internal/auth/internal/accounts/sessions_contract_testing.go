//go:build testing

package accounts

import (
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// SessionsContractSuite is the behaviour every Sessions shares. Embed it and set NewSessions.
type SessionsContractSuite struct {
	suite.Suite

	// NewSessions answers an empty store.
	NewSessions func() Sessions

	sessions Sessions
}

func (s *SessionsContractSuite) SetupTest() {
	s.sessions = s.NewSessions()
}

var (
	contractStart   = time.Date(2026, 9, 15, 12, 0, 0, 123_000_000, time.UTC)
	contractAccount = uuid.MustParse("01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f")
	contractHash    = HashOf("a-token")
)

func (s *SessionsContractSuite) find(hash []byte) (Session, bool) {
	session, found, err := s.sessions.FindSession(s.T().Context(), hash)
	s.Require().NoError(err)
	return session, found
}

func (s *SessionsContractSuite) TestAnUnknownTokenFindsNothing() {
	_, found := s.find(contractHash)
	s.False(found)
}

func (s *SessionsContractSuite) TestACreatedGuestIsFoundByItsTokenHash() {
	s.Require().NoError(s.sessions.CreateGuest(s.T().Context(), contractAccount, contractHash, contractStart.Add(time.Hour), contractStart))

	session, found := s.find(contractHash)

	s.True(found)
	s.Equal(Session{Account: contractAccount, ExtendedAt: contractStart, ExpiresAt: contractStart.Add(time.Hour)}, session)
}

func (s *SessionsContractSuite) TestTwoGuestsAreFoundApart() {
	other := uuid.MustParse("01926c6e-0000-7000-8000-000000000001")
	otherHash := HashOf("another-token")

	s.Require().NoError(s.sessions.CreateGuest(s.T().Context(), contractAccount, contractHash, contractStart.Add(time.Hour), contractStart))
	s.Require().NoError(s.sessions.CreateGuest(s.T().Context(), other, otherHash, contractStart.Add(time.Hour), contractStart))

	first, _ := s.find(contractHash)
	second, _ := s.find(otherHash)
	s.Equal(contractAccount, first.Account)
	s.Equal(other, second.Account)
}

func (s *SessionsContractSuite) TestExtendingMovesTheExpiry() {
	s.Require().NoError(s.sessions.CreateGuest(s.T().Context(), contractAccount, contractHash, contractStart.Add(time.Hour), contractStart))

	later := contractStart.Add(30 * time.Minute)
	s.Require().NoError(s.sessions.ExtendSession(s.T().Context(), contractHash, later.Add(time.Hour), later))

	session, _ := s.find(contractHash)
	s.Equal(Session{Account: contractAccount, ExtendedAt: later, ExpiresAt: later.Add(time.Hour)}, session)
}

func (s *SessionsContractSuite) TestExtendingAnUnknownSessionFails() {
	s.Error(s.sessions.ExtendSession(s.T().Context(), contractHash, contractStart.Add(time.Hour), contractStart))
}

func (s *SessionsContractSuite) TestATakenTokenHashCreatesNoSecondGuest() {
	s.Require().NoError(s.sessions.CreateGuest(s.T().Context(), contractAccount, contractHash, contractStart.Add(time.Hour), contractStart))

	other := uuid.MustParse("01926c6e-0000-7000-8000-000000000001")
	s.Require().Error(s.sessions.CreateGuest(s.T().Context(), other, contractHash, contractStart.Add(2*time.Hour), contractStart))

	session, _ := s.find(contractHash)
	s.Equal(contractAccount, session.Account, "the first guest keeps its session")
}
