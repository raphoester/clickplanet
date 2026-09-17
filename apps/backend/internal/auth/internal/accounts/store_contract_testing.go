//go:build testing

package accounts

import (
	"errors"
	"time"

	"github.com/stretchr/testify/suite"
)

// StoreContractSuite is the behaviour every Store shares. Embed it and set NewStore.
type StoreContractSuite struct {
	suite.Suite

	// NewStore answers an empty store.
	NewStore func() Store

	store Store
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

var (
	contractStart    = time.Date(2026, 9, 15, 12, 0, 0, 123_000_000, time.UTC)
	contractLifetime = Lifetime{GuestTTL: time.Hour, LinkedTTL: 30 * time.Minute, ExtendEvery: time.Minute}
	contractClaim    = Claim{Subject: "google-user", Email: "player@example.com", EmailVerified: true}
)

func (s *StoreContractSuite) guest(account byte, token string) *Session {
	return GuestSession(AccountID{15: account}, TokenOf(token), contractLifetime, contractStart)
}

func (s *StoreContractSuite) createGuest(account byte, token string) *Session {
	guest := s.guest(account, token)
	s.Require().NoError(s.store.CreateGuest(s.T().Context(), guest))
	return guest
}

// signIn links provider's subject to account, a new one unless a guest already holds it.
func (s *StoreContractSuite) signIn(account byte, provider string, subject string, token string) (*Identity, *Session) {
	_, err := s.store.Account(s.T().Context(), AccountID{15: account})
	identity := NewIdentity(provider, Claim{Subject: subject}, AccountID{15: account}, contractStart)
	session := LinkedSession(identity.Account, TokenOf(token), contractLifetime, contractStart)
	s.Require().NoError(s.store.SaveSignIn(s.T().Context(), SignIn{
		NewAccount: errors.Is(err, ErrAccountNotFound), Identity: identity, Session: session,
	}))
	return identity, session
}

func (s *StoreContractSuite) stored(tokenHash TokenHash) *Session {
	session, err := s.store.Session(s.T().Context(), tokenHash)
	s.Require().NoError(err)
	return session
}

func (s *StoreContractSuite) account(account byte) *Account {
	found, err := s.store.Account(s.T().Context(), AccountID{15: account})
	s.Require().NoError(err)
	return found
}

func (s *StoreContractSuite) TestAnUnknownTokenIsNotFound() {
	session, err := s.store.Session(s.T().Context(), TokenOf("unknown").Hash)

	s.Require().ErrorIs(err, ErrSessionNotFound)
	s.Nil(session)
}

func (s *StoreContractSuite) TestACreatedGuestIsFoundByItsTokenHash() {
	guest := s.createGuest(1, "a-token")

	s.Equal(guest, s.stored(guest.TokenHash))
	s.Equal(&Account{ID: guest.Account, Identities: []Identity{}}, s.account(1))
}

func (s *StoreContractSuite) TestTwoGuestsAreFoundApart() {
	first, second := s.createGuest(1, "a-token"), s.createGuest(2, "another-token")

	s.Equal(first, s.stored(first.TokenHash))
	s.Equal(second, s.stored(second.TokenHash))
}

func (s *StoreContractSuite) TestASavedSessionKeepsItsNewExpiry() {
	guest := s.createGuest(1, "a-token")

	extended := guest.Extended(contractStart.Add(30*time.Minute), contractLifetime)
	s.Require().NoError(s.store.SaveSession(s.T().Context(), extended))

	s.Equal(extended, s.stored(guest.TokenHash))
}

func (s *StoreContractSuite) TestSavingAnUnknownSessionIsNotFound() {
	s.ErrorIs(s.store.SaveSession(s.T().Context(), s.guest(1, "a-token")), ErrSessionNotFound)
}

func (s *StoreContractSuite) TestATakenTokenHashCreatesNoSecondGuest() {
	first := s.createGuest(1, "a-token")

	s.Require().Error(s.store.CreateGuest(s.T().Context(), s.guest(2, "a-token")))

	s.Equal(first, s.stored(first.TokenHash), "the first guest keeps its session")
	_, err := s.store.Account(s.T().Context(), AccountID{15: 2})
	s.ErrorIs(err, ErrAccountNotFound, "a guest whose session fails to insert leaves no account")
}

func (s *StoreContractSuite) TestAnUnknownAccountIsNotFound() {
	account, err := s.store.Account(s.T().Context(), AccountID{15: 9})

	s.Require().ErrorIs(err, ErrAccountNotFound)
	s.Nil(account)
}

func (s *StoreContractSuite) TestAnUnknownIdentityIsNotFound() {
	identity, err := s.store.Identity(s.T().Context(), "google", "nobody")

	s.Require().ErrorIs(err, ErrIdentityNotFound)
	s.Nil(identity)
}

func (s *StoreContractSuite) TestASignInToANewAccountCreatesItLinked() {
	identity := NewIdentity("google", contractClaim, AccountID{15: 1}, contractStart)
	session := LinkedSession(identity.Account, TokenOf("a-token"), contractLifetime, contractStart)

	s.Require().NoError(s.store.SaveSignIn(s.T().Context(), SignIn{NewAccount: true, Identity: identity, Session: session}))

	s.Equal(&Account{ID: identity.Account, Identities: []Identity{*identity}}, s.account(1))
	s.Equal(session, s.stored(session.TokenHash))
	found, err := s.store.Identity(s.T().Context(), "google", "google-user")
	s.Require().NoError(err)
	s.Equal(identity, found)
}

func (s *StoreContractSuite) TestLinkingAGuestReplacesItsSession() {
	guest := s.createGuest(1, "guest-token")
	identity := NewIdentity("discord", Claim{Subject: "discord-user"}, guest.Account, contractStart.Add(time.Minute))
	session := LinkedSession(guest.Account, TokenOf("linked-token"), contractLifetime, contractStart.Add(time.Minute))

	s.Require().NoError(s.store.SaveSignIn(s.T().Context(), SignIn{Identity: identity, Session: session, Replaces: guest.TokenHash}))

	_, err := s.store.Session(s.T().Context(), guest.TokenHash)
	s.Require().ErrorIs(err, ErrSessionNotFound)
	s.Equal(session, s.stored(session.TokenHash))
	s.Equal([]string{"discord"}, s.account(1).Providers())
}

func (s *StoreContractSuite) TestProvidersAreListedOldestLinkFirst() {
	s.signIn(1, "google", "google-user", "first-token")
	later := NewIdentity("discord", Claim{Subject: "discord-user"}, AccountID{15: 1}, contractStart.Add(time.Hour))
	session := LinkedSession(later.Account, TokenOf("second-token"), contractLifetime, contractStart.Add(time.Hour))
	s.Require().NoError(s.store.SaveSignIn(s.T().Context(), SignIn{Identity: later, Session: session}))

	s.Equal([]string{"google", "discord"}, s.account(1).Providers())
}

func (s *StoreContractSuite) TestASignInToAKnownIdentityOnlyStartsASession() {
	s.signIn(1, "google", "google-user", "first-token")
	guest := s.createGuest(2, "guest-token")
	session := LinkedSession(AccountID{15: 1}, TokenOf("second-token"), contractLifetime, contractStart)

	s.Require().NoError(s.store.SaveSignIn(s.T().Context(), SignIn{Session: session, Replaces: guest.TokenHash}))

	s.Equal(session, s.stored(session.TokenHash))
	s.Equal([]string{"google"}, s.account(1).Providers())
	s.Equal(&Account{ID: guest.Account, Identities: []Identity{}}, s.account(2), "the guest is left as it was")
}

func (s *StoreContractSuite) TestAnIdentityLinkedTwiceIsTakenAndWritesNothing() {
	s.signIn(1, "google", "google-user", "first-token")
	identity := NewIdentity("google", Claim{Subject: "google-user"}, AccountID{15: 2}, contractStart)
	session := LinkedSession(identity.Account, TokenOf("second-token"), contractLifetime, contractStart)

	err := s.store.SaveSignIn(s.T().Context(), SignIn{NewAccount: true, Identity: identity, Session: session})

	s.Require().ErrorIs(err, ErrIdentityTaken)
	_, err = s.store.Account(s.T().Context(), identity.Account)
	s.Require().ErrorIs(err, ErrAccountNotFound)
	_, err = s.store.Session(s.T().Context(), session.TokenHash)
	s.ErrorIs(err, ErrSessionNotFound)
}

func (s *StoreContractSuite) TestASessionOfALinkedAccountIsLinked() {
	guest := s.createGuest(1, "guest-token")
	s.Require().False(guest.Linked)

	s.signIn(1, "google", "google-user", "linked-token")

	s.True(s.stored(guest.TokenHash).Linked, "every session of the account reads linked, and extends as one")
}

func (s *StoreContractSuite) TestSigningOutDeletesOnlyThatSession() {
	_, first := s.signIn(1, "google", "google-user", "first-token")
	second := LinkedSession(AccountID{15: 1}, TokenOf("second-token"), contractLifetime, contractStart)
	s.Require().NoError(s.store.SaveSignIn(s.T().Context(), SignIn{Session: second}))

	s.Require().NoError(s.store.DeleteSession(s.T().Context(), first.TokenHash))
	s.Require().NoError(s.store.DeleteSession(s.T().Context(), first.TokenHash), "twice is not an error")

	_, err := s.store.Session(s.T().Context(), first.TokenHash)
	s.Require().ErrorIs(err, ErrSessionNotFound)
	s.Equal(second, s.stored(second.TokenHash))
}

func (s *StoreContractSuite) TestSigningOutEverywhereDeletesEverySessionOfTheAccountAndNoOther() {
	_, first := s.signIn(1, "google", "google-user", "first-token")
	second := LinkedSession(AccountID{15: 1}, TokenOf("second-token"), contractLifetime, contractStart)
	s.Require().NoError(s.store.SaveSignIn(s.T().Context(), SignIn{Session: second}))
	other := s.createGuest(2, "other-token")

	s.Require().NoError(s.store.DeleteSessions(s.T().Context(), AccountID{15: 1}))

	for _, gone := range []TokenHash{first.TokenHash, second.TokenHash} {
		_, err := s.store.Session(s.T().Context(), gone)
		s.Require().ErrorIs(err, ErrSessionNotFound)
	}
	s.Equal(other, s.stored(other.TokenHash))
	s.Equal([]string{"google"}, s.account(1).Providers(), "the account stays")
}

func (s *StoreContractSuite) TestADeletedAccountTakesItsIdentitiesAndSessions() {
	_, session := s.signIn(1, "google", "google-user", "a-token")
	other := s.createGuest(2, "other-token")

	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), AccountID{15: 1}))
	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), AccountID{15: 1}), "twice is not an error")

	_, err := s.store.Account(s.T().Context(), AccountID{15: 1})
	s.Require().ErrorIs(err, ErrAccountNotFound)
	_, err = s.store.Session(s.T().Context(), session.TokenHash)
	s.Require().ErrorIs(err, ErrSessionNotFound)
	_, err = s.store.Identity(s.T().Context(), "google", "google-user")
	s.Require().ErrorIs(err, ErrIdentityNotFound)
	s.Equal(other, s.stored(other.TokenHash))
}

func (s *StoreContractSuite) TestThePruneDeletesIdleGuestsOnly() {
	idle := s.createGuest(1, "idle-token")
	s.signIn(2, "google", "google-user", "linked-token")
	recent := s.createGuest(3, "recent-token")
	s.Require().NoError(s.store.SaveSession(s.T().Context(), recent.Extended(contractStart.Add(time.Hour), contractLifetime)))

	pruned, err := s.store.PruneGuests(s.T().Context(), contractStart.Add(time.Minute), 100)

	s.Require().NoError(err)
	s.Equal([]AccountID{idle.Account}, pruned)
	_, err = s.store.Account(s.T().Context(), idle.Account)
	s.Require().ErrorIs(err, ErrAccountNotFound)
	_, err = s.store.Session(s.T().Context(), idle.TokenHash)
	s.Require().ErrorIs(err, ErrSessionNotFound)
	s.account(2)
	s.account(3)
}

func (s *StoreContractSuite) TestThePruneStopsAtItsLimit() {
	for account := range byte(3) {
		s.createGuest(account+1, string(rune('a'+account)))
	}

	pruned, err := s.store.PruneGuests(s.T().Context(), contractStart.Add(time.Minute), 2)
	s.Require().NoError(err)
	s.Len(pruned, 2)

	pruned, err = s.store.PruneGuests(s.T().Context(), contractStart.Add(time.Minute), 2)
	s.Require().NoError(err)
	s.Len(pruned, 1)
}
