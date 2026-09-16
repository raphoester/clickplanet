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
	contractStart    = time.Date(2026, 9, 15, 12, 0, 0, 123_000_000, time.UTC)
	contractLifetime = Lifetime{GuestTTL: time.Hour, ExtendEvery: time.Minute}
)

func (s *SessionsContractSuite) guest(account byte, token string) *Session {
	return StartGuest(uuid.UUID{15: account}, TokenOf(token), contractLifetime, contractStart)
}

func (s *SessionsContractSuite) find(tokenHash []byte) *Session {
	session, err := s.sessions.FindSession(s.T().Context(), tokenHash)
	s.Require().NoError(err)
	return session
}

func (s *SessionsContractSuite) TestAnUnknownTokenIsNotFound() {
	session, err := s.sessions.FindSession(s.T().Context(), TokenOf("unknown").Hash)

	s.Require().ErrorIs(err, ErrSessionNotFound)
	s.Nil(session)
}

func (s *SessionsContractSuite) TestACreatedGuestIsFoundByItsTokenHash() {
	guest := s.guest(1, "a-token")
	s.Require().NoError(s.sessions.CreateGuest(s.T().Context(), guest))

	s.Equal(guest, s.find(guest.TokenHash))
}

func (s *SessionsContractSuite) TestTwoGuestsAreFoundApart() {
	first, second := s.guest(1, "a-token"), s.guest(2, "another-token")
	s.Require().NoError(s.sessions.CreateGuest(s.T().Context(), first))
	s.Require().NoError(s.sessions.CreateGuest(s.T().Context(), second))

	s.Equal(first, s.find(first.TokenHash))
	s.Equal(second, s.find(second.TokenHash))
}

func (s *SessionsContractSuite) TestASavedSessionKeepsItsNewExpiry() {
	guest := s.guest(1, "a-token")
	s.Require().NoError(s.sessions.CreateGuest(s.T().Context(), guest))

	s.Require().True(guest.ExtendIfDue(contractStart.Add(30*time.Minute), contractLifetime))
	s.Require().NoError(s.sessions.SaveSession(s.T().Context(), guest))

	s.Equal(guest, s.find(guest.TokenHash))
}

func (s *SessionsContractSuite) TestSavingAnUnknownSessionIsNotFound() {
	s.ErrorIs(s.sessions.SaveSession(s.T().Context(), s.guest(1, "a-token")), ErrSessionNotFound)
}

func (s *SessionsContractSuite) TestATakenTokenHashCreatesNoSecondGuest() {
	first := s.guest(1, "a-token")
	s.Require().NoError(s.sessions.CreateGuest(s.T().Context(), first))

	s.Require().Error(s.sessions.CreateGuest(s.T().Context(), s.guest(2, "a-token")))

	s.Equal(first, s.find(first.TokenHash), "the first guest keeps its session")
}
