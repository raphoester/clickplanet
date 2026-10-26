package bans_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func newBook(t *testing.T, clock cptime.Clock) (book, *bans.MemoryStore, *bans.MemoryStore) {
	t.Helper()

	scopes, accounts := bans.NewMemoryStore(), bans.NewMemoryStore()
	return book{t: t, inner: bans.NewBook(config(), pardon, clock, scopes, accounts, cpatomic.FakeTransactioner{}, &bans.MemoryEnforcer{})}, scopes, accounts
}

func TestAFlaggedGuestIsBannedOnItsAccountAndItsScope(t *testing.T) {
	book, _, _ := newBook(t, newClock())

	_, accepted := book.Flag(bans.Caller{Scope: "1.2.3.4", Account: "guest"})
	require.True(t, accepted)

	assert.True(t, book.Banned(bans.Caller{Scope: "1.2.3.4", Account: "a-fresh-cookie"}),
		"a new cookie on the same scope is still dropped")
	assert.True(t, book.Banned(bans.Caller{Scope: "5.6.7.8", Account: "guest"}),
		"the same account on another scope is still dropped")
	assert.False(t, book.Banned(bans.Caller{Scope: "5.6.7.8", Account: "someone-else"}))
	assert.Equal(t, 2, book.Flagged(), "one guest, banned twice")
}

func TestAFlaggedSignedInAccountLeavesItsScopeAlone(t *testing.T) {
	book, _, _ := newBook(t, newClock())

	_, accepted := book.Flag(bans.Caller{Scope: "campus", Account: "player", SignedIn: true})
	require.True(t, accepted)

	assert.True(t, book.Banned(bans.Caller{Scope: "home", Account: "player", SignedIn: true}))
	assert.False(t, book.Banned(bans.Caller{Scope: "campus", Account: "classmate", SignedIn: true}),
		"the other players behind the campus address are not banned with it")
}

func TestAFlagWithNoAccountBansTheScope(t *testing.T) {
	book, _, _ := newBook(t, newClock())

	_, accepted := book.Flag(bans.Caller{Scope: "1.2.3.4"})
	require.True(t, accepted)

	assert.True(t, book.Banned(bans.Caller{Scope: "1.2.3.4"}))
	assert.True(t, book.Banned(bans.Caller{Scope: "1.2.3.4", Account: "guest"}))
}

func TestAnOperatorBanOnAnAccountBansTheAccountAlone(t *testing.T) {
	clock := newClock()
	book, _, _ := newBook(t, clock)

	sentence := book.Ban(bans.Caller{Account: "guest"}, 0)
	assert.Equal(t, 1, sentence.Offence)
	assert.Equal(t, clock.Now().Add(time.Hour), sentence.Until)

	assert.True(t, book.Banned(bans.Caller{Scope: "1.2.3.4", Account: "guest"}))
	assert.False(t, book.Banned(bans.Caller{Scope: "1.2.3.4"}))
}

func TestTheSentenceIsTheBanThatEndsLast(t *testing.T) {
	clock := newClock()
	book, _, _ := newBook(t, clock)

	book.Ban(bans.Caller{Scope: "1.2.3.4"}, time.Hour)
	book.Ban(bans.Caller{Account: "guest"}, 2*time.Hour)

	sentence, running := book.Sentence(bans.Caller{Scope: "1.2.3.4", Account: "guest"})
	require.True(t, running)
	assert.Equal(t, clock.Now().Add(2*time.Hour), sentence.Until)

	_, running = book.Sentence(bans.Caller{Scope: "5.6.7.8", Account: "someone-else"})
	assert.False(t, running)
}

func TestAccountBansAreKeptApart(t *testing.T) {
	clock := newClock()
	before, scopes, accounts := newBook(t, clock)

	before.Flag(bans.Caller{Scope: "1.2.3.4", Account: "guest"})

	assert.Contains(t, scopes.Stored(), "1.2.3.4")
	assert.NotContains(t, scopes.Stored(), "guest")
	assert.Contains(t, accounts.Stored(), "guest")

	after := book{t: t, inner: bans.NewBook(config(), pardon, clock, scopes, accounts, cpatomic.FakeTransactioner{}, &bans.MemoryEnforcer{})}
	assert.True(t, after.Banned(bans.Caller{Scope: "5.6.7.8", Account: "guest"}))
}

func TestAnUnbanLiftsOnlyTheBanItNames(t *testing.T) {
	book, _, _ := newBook(t, newClock())
	guest := bans.Caller{Scope: "1.2.3.4", Account: "guest"}

	book.Flag(guest)

	book.Unban(bans.Caller{Account: "guest"})
	assert.True(t, book.Banned(guest), "the guest's scope is still banned")
	assert.False(t, book.Banned(bans.Caller{Scope: "5.6.7.8", Account: "guest"}))

	book.Unban(bans.Caller{Scope: "1.2.3.4"})
	assert.False(t, book.Banned(guest))
	assert.Equal(t, 0, book.Flagged())
}

func TestTheLastSentenceOfAnAccountIsReadAfterItsBanEnded(t *testing.T) {
	clock := newClock()
	book, _, accounts := newBook(t, clock)
	player := bans.Caller{Account: "player"}

	banned := book.Ban(player, 0)
	clock.Advance(2 * time.Hour)

	_, running := book.Sentence(player)
	require.False(t, running)

	last, found := book.LastSentence(player)
	require.True(t, found)
	assert.Equal(t, banned, last)

	book.Unban(player)
	_, found = book.LastSentence(player)
	assert.False(t, found)
	assert.Equal(t, 0, accounts.Stored()["player"].Offences)
}

func TestAGuestIsPardonedOnceItsAccountAndItsScopeAreUnbanned(t *testing.T) {
	book, _, _ := newBook(t, newClock())
	guest := bans.Caller{Scope: "1.2.3.4", Account: "guest"}

	book.Flag(guest)

	book.Unban(bans.Caller{Account: "guest"})
	assert.False(t, book.Pardoned(guest), "its scope is not pardoned")

	book.Unban(bans.Caller{Scope: "1.2.3.4"})
	assert.True(t, book.Pardoned(guest))
	_, accepted := book.Flag(guest)
	assert.False(t, accepted)
	assert.False(t, book.Banned(guest))
}

func TestASignedInAccountIsPardonedOnItsAccountAlone(t *testing.T) {
	book, _, _ := newBook(t, newClock())
	player := bans.Caller{Scope: "campus", Account: "player", SignedIn: true}

	book.Flag(player)
	book.Unban(bans.Caller{Account: "player"})

	assert.True(t, book.Pardoned(player))
	assert.False(t, book.Pardoned(bans.Caller{Scope: "campus", Account: "classmate", SignedIn: true}))
}

func TestThePardonIsTheOneThatEndsLast(t *testing.T) {
	clock := newClock()
	book, _, _ := newBook(t, clock)

	book.Ban(bans.Caller{Scope: "1.2.3.4"}, time.Hour)
	book.Ban(bans.Caller{Account: "guest"}, time.Hour)
	book.Unban(bans.Caller{Scope: "1.2.3.4"})
	clock.Advance(time.Minute)
	book.Unban(bans.Caller{Account: "guest"})

	pardoned, found := book.Pardon(bans.Caller{Scope: "1.2.3.4", Account: "guest"})
	require.True(t, found)
	assert.Equal(t, clock.Now().Add(pardon), pardoned.Until)

	pardoned, found = book.Pardon(bans.Caller{Scope: "1.2.3.4", Account: "someone-else"})
	require.True(t, found)
	assert.Equal(t, clock.Now().Add(pardon-time.Minute), pardoned.Until)

	_, found = book.Pardon(bans.Caller{Scope: "5.6.7.8", Account: "someone-else"})
	assert.False(t, found)
}
