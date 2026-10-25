//go:build testing

package accounts

import (
	"errors"
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
	contractStart    = time.Date(2026, 9, 15, 12, 0, 0, 123_000_000, time.UTC)
	contractLifetime = Lifetime{GuestTTL: time.Hour, LinkedTTL: 30 * time.Minute, ExtendEvery: time.Minute}
	contractClaim    = Claim{subject: "google-user", email: "player@example.com", emailVerified: true}
)

func (s *StoreContractSuite) guest(account byte, token string) *Session {
	return GuestSession(AccountID{15: account}, TokenOf(token), contractLifetime, contractStart)
}

func (s *StoreContractSuite) createGuest(account byte, token string) *Session {
	guest := s.guest(account, token)
	s.Require().NoError(s.store.CreateGuest(s.T().Context(), guest))
	return guest
}

func (s *StoreContractSuite) signIn(account byte, provider string, subject string, token string) (*Identity, *Session) {
	_, err := s.store.Account(s.T().Context(), AccountID{15: account})
	identity := NewIdentity(provider, Claim{subject: subject}, AccountID{15: account}, contractStart)
	session := LinkedSession(identity.account, TokenOf(token), contractLifetime, contractStart)
	s.Require().NoError(s.store.SaveSignIn(s.T().Context(), SignIn{
		newAccount: errors.Is(err, ErrAccountNotFound), identity: identity, session: session,
	}))
	return identity, session
}

func (s *StoreContractSuite) signInWith(account byte, provider string, claim Claim, token string, at time.Time) {
	_, err := s.store.Account(s.T().Context(), AccountID{15: account})
	identity := NewIdentity(provider, claim, AccountID{15: account}, at)
	s.Require().NoError(s.store.SaveSignIn(s.T().Context(), SignIn{
		newAccount: errors.Is(err, ErrAccountNotFound), identity: identity,
		session: LinkedSession(identity.account, TokenOf(token), contractLifetime, at),
	}))
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
	session, err := s.store.Session(s.T().Context(), TokenOf("unknown").hash)

	s.Require().ErrorIs(err, ErrSessionNotFound)
	s.Nil(session)
}

func (s *StoreContractSuite) TestACreatedGuestIsFoundByItsTokenHash() {
	guest := s.createGuest(1, "a-token")

	s.Equal(guest, s.stored(guest.tokenHash))
	s.Equal(&Account{id: guest.account, createdAt: contractStart, identities: []Identity{}}, s.account(1))
}

func (s *StoreContractSuite) TestTwoGuestsAreFoundApart() {
	first, second := s.createGuest(1, "a-token"), s.createGuest(2, "another-token")

	s.Equal(first, s.stored(first.tokenHash))
	s.Equal(second, s.stored(second.tokenHash))
}

func (s *StoreContractSuite) TestASavedSessionKeepsItsNewExpiry() {
	guest := s.createGuest(1, "a-token")

	extended := guest.Extended(contractStart.Add(30*time.Minute), contractLifetime)
	s.Require().NoError(s.store.SaveSession(s.T().Context(), extended))

	s.Equal(extended, s.stored(guest.tokenHash))
}

func (s *StoreContractSuite) TestSavingAnUnknownSessionIsNotFound() {
	s.ErrorIs(s.store.SaveSession(s.T().Context(), s.guest(1, "a-token")), ErrSessionNotFound)
}

func (s *StoreContractSuite) TestATakenTokenHashCreatesNoSecondGuest() {
	first := s.createGuest(1, "a-token")

	s.Require().Error(s.store.CreateGuest(s.T().Context(), s.guest(2, "a-token")))

	s.Equal(first, s.stored(first.tokenHash), "the first guest keeps its session")
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
	session := LinkedSession(identity.account, TokenOf("a-token"), contractLifetime, contractStart)

	s.Require().NoError(s.store.SaveSignIn(s.T().Context(), SignIn{newAccount: true, identity: identity, session: session}))

	s.Equal(&Account{id: identity.account, createdAt: contractStart, identities: []Identity{*identity}}, s.account(1))
	s.Equal(session, s.stored(session.tokenHash))
	found, err := s.store.Identity(s.T().Context(), "google", "google-user")
	s.Require().NoError(err)
	s.Equal(identity, found)
}

func (s *StoreContractSuite) TestLinkingAGuestReplacesItsSession() {
	guest := s.createGuest(1, "guest-token")
	identity := NewIdentity("discord", Claim{subject: "discord-user"}, guest.account, contractStart.Add(time.Minute))
	session := LinkedSession(guest.account, TokenOf("linked-token"), contractLifetime, contractStart.Add(time.Minute))

	s.Require().NoError(s.store.SaveSignIn(s.T().Context(), SignIn{identity: identity, session: session, replaces: guest.tokenHash}))

	_, err := s.store.Session(s.T().Context(), guest.tokenHash)
	s.Require().ErrorIs(err, ErrSessionNotFound)
	s.Equal(session, s.stored(session.tokenHash))
	s.Equal([]string{"discord"}, s.account(1).Providers())
	s.Equal(contractStart, s.account(1).createdAt, "a linked guest keeps the day it was made")
}

func (s *StoreContractSuite) TestProvidersAreListedOldestLinkFirst() {
	s.signIn(1, "google", "google-user", "first-token")
	later := NewIdentity("discord", Claim{subject: "discord-user"}, AccountID{15: 1}, contractStart.Add(time.Hour))
	session := LinkedSession(later.account, TokenOf("second-token"), contractLifetime, contractStart.Add(time.Hour))
	s.Require().NoError(s.store.SaveSignIn(s.T().Context(), SignIn{identity: later, session: session}))

	s.Equal([]string{"google", "discord"}, s.account(1).Providers())
}

func (s *StoreContractSuite) TestProvidersLinkedAtTheSameTimeAreListedByName() {
	s.signIn(1, "google", "google-user", "first-token")
	same := NewIdentity("discord", Claim{subject: "discord-user"}, AccountID{15: 1}, contractStart)
	session := LinkedSession(same.account, TokenOf("second-token"), contractLifetime, contractStart)
	s.Require().NoError(s.store.SaveSignIn(s.T().Context(), SignIn{identity: same, session: session}))

	s.Equal([]string{"discord", "google"}, s.account(1).Providers())
}

func (s *StoreContractSuite) TestASignInToAKnownIdentityOnlyStartsASession() {
	s.signIn(1, "google", "google-user", "first-token")
	guest := s.createGuest(2, "guest-token")
	session := LinkedSession(AccountID{15: 1}, TokenOf("second-token"), contractLifetime, contractStart)

	s.Require().NoError(s.store.SaveSignIn(s.T().Context(), SignIn{session: session, replaces: guest.tokenHash}))

	s.Equal(session, s.stored(session.tokenHash))
	s.Equal([]string{"google"}, s.account(1).Providers())
	s.Equal(&Account{id: guest.account, createdAt: contractStart, identities: []Identity{}}, s.account(2), "the guest is left as it was")
}

func (s *StoreContractSuite) TestAnIdentityLinkedTwiceIsTakenAndWritesNothing() {
	s.signIn(1, "google", "google-user", "first-token")
	identity := NewIdentity("google", Claim{subject: "google-user"}, AccountID{15: 2}, contractStart)
	session := LinkedSession(identity.account, TokenOf("second-token"), contractLifetime, contractStart)

	err := s.store.SaveSignIn(s.T().Context(), SignIn{newAccount: true, identity: identity, session: session})

	s.Require().ErrorIs(err, ErrIdentityTaken)
	_, err = s.store.Account(s.T().Context(), identity.account)
	s.Require().ErrorIs(err, ErrAccountNotFound)
	_, err = s.store.Session(s.T().Context(), session.tokenHash)
	s.ErrorIs(err, ErrSessionNotFound)
}

func (s *StoreContractSuite) TestASessionOfALinkedAccountIsLinked() {
	guest := s.createGuest(1, "guest-token")
	s.Require().False(guest.linked)

	s.signIn(1, "google", "google-user", "linked-token")

	s.True(s.stored(guest.tokenHash).linked, "every session of the account reads linked, and extends as one")
}

func (s *StoreContractSuite) TestSigningOutDeletesOnlyThatSession() {
	_, first := s.signIn(1, "google", "google-user", "first-token")
	second := LinkedSession(AccountID{15: 1}, TokenOf("second-token"), contractLifetime, contractStart)
	s.Require().NoError(s.store.SaveSignIn(s.T().Context(), SignIn{session: second}))

	s.Require().NoError(s.store.DeleteSession(s.T().Context(), first.tokenHash))
	s.Require().NoError(s.store.DeleteSession(s.T().Context(), first.tokenHash), "twice is not an error")

	_, err := s.store.Session(s.T().Context(), first.tokenHash)
	s.Require().ErrorIs(err, ErrSessionNotFound)
	s.Equal(second, s.stored(second.tokenHash))
}

func (s *StoreContractSuite) TestSigningOutEverywhereDeletesEverySessionOfTheAccountAndNoOther() {
	_, first := s.signIn(1, "google", "google-user", "first-token")
	second := LinkedSession(AccountID{15: 1}, TokenOf("second-token"), contractLifetime, contractStart)
	s.Require().NoError(s.store.SaveSignIn(s.T().Context(), SignIn{session: second}))
	other := s.createGuest(2, "other-token")

	s.Require().NoError(s.store.DeleteSessions(s.T().Context(), AccountID{15: 1}))

	for _, gone := range []TokenHash{first.tokenHash, second.tokenHash} {
		_, err := s.store.Session(s.T().Context(), gone)
		s.Require().ErrorIs(err, ErrSessionNotFound)
	}
	s.Equal(other, s.stored(other.tokenHash))
	s.Equal([]string{"google"}, s.account(1).Providers(), "the account stays")
}

func (s *StoreContractSuite) TestADeletedAccountTakesItsIdentitiesAndSessions() {
	_, session := s.signIn(1, "google", "google-user", "a-token")
	other := s.createGuest(2, "other-token")

	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), AccountID{15: 1}))
	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), AccountID{15: 1}), "twice is not an error")

	_, err := s.store.Account(s.T().Context(), AccountID{15: 1})
	s.Require().ErrorIs(err, ErrAccountNotFound)
	_, err = s.store.Session(s.T().Context(), session.tokenHash)
	s.Require().ErrorIs(err, ErrSessionNotFound)
	_, err = s.store.Identity(s.T().Context(), "google", "google-user")
	s.Require().ErrorIs(err, ErrIdentityNotFound)
	s.Equal(other, s.stored(other.tokenHash))
}

func (s *StoreContractSuite) TestThePruneDeletesIdleGuestsOnly() {
	idle := s.createGuest(1, "idle-token")
	s.signIn(2, "google", "google-user", "linked-token")
	recent := s.createGuest(3, "recent-token")
	s.Require().NoError(s.store.SaveSession(s.T().Context(), recent.Extended(contractStart.Add(time.Hour), contractLifetime)))

	pruned, err := s.store.PruneGuests(s.T().Context(), contractStart.Add(time.Minute), 100)

	s.Require().NoError(err)
	s.Equal([]AccountID{idle.account}, pruned)
	_, err = s.store.Account(s.T().Context(), idle.account)
	s.Require().ErrorIs(err, ErrAccountNotFound)
	_, err = s.store.Session(s.T().Context(), idle.tokenHash)
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

func (s *StoreContractSuite) TestAnAccountIsFoundByAVerifiedAddressOfAnyIdentityIgnoringCase() {
	s.signInWith(1, "discord", Claim{subject: "d", email: "Player@Example.com", emailVerified: true}, "token-1", contractStart)

	found, err := s.store.AccountOfEmail(s.T().Context(), "player@example.COM")

	s.Require().NoError(err)
	s.Equal(AccountID{15: 1}, found.id)
	s.Equal([]string{"discord"}, found.Providers())
}

func (s *StoreContractSuite) TestAnUnverifiedOrUnknownAddressFindsNoAccount() {
	s.signInWith(1, "discord", Claim{subject: "d", email: "player@example.com"}, "token-1", contractStart)

	for _, address := range []string{"player@example.com", "nobody@example.com", ""} {
		_, err := s.store.AccountOfEmail(s.T().Context(), address)

		s.ErrorIs(err, ErrAccountNotFound, address)
	}
}

func (s *StoreContractSuite) TestTheOldestLinkOwnsAnAddressTwoAccountsHold() {
	s.signInWith(2, "google", Claim{subject: "g", email: "player@example.com", emailVerified: true}, "token-2", contractStart.Add(time.Minute))
	s.signInWith(1, "discord", Claim{subject: "d", email: "player@example.com", emailVerified: true}, "token-1", contractStart)

	found, err := s.store.AccountOfEmail(s.T().Context(), "player@example.com")

	s.Require().NoError(err)
	s.Equal(AccountID{15: 1}, found.id)
}
