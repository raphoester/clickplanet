package shadowban_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/shadowban"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func newBans(t *testing.T, clock cptime.Clock) (*shadowban.Bans, *shadowban.MemoryPersistence, *shadowban.MemoryPersistence) {
	t.Helper()

	scopes, accounts := shadowban.NewMemoryPersistence(), shadowban.NewMemoryPersistence()
	return shadowban.NewBans(config(), clock, scopes, accounts, failOnStateError(t)), scopes, accounts
}

func TestAFlaggedGuestIsBannedOnItsAccountAndItsScope(t *testing.T) {
	bans, _, _ := newBans(t, newClock())

	_, accepted := bans.Flag(shadowban.Caller{Scope: "1.2.3.4", Account: "guest"})
	require.True(t, accepted)

	assert.True(t, bans.Banned(shadowban.Caller{Scope: "1.2.3.4", Account: "a-fresh-cookie"}),
		"a new cookie on the same scope is still dropped")
	assert.True(t, bans.Banned(shadowban.Caller{Scope: "5.6.7.8", Account: "guest"}),
		"the same account on another scope is still dropped")
	assert.False(t, bans.Banned(shadowban.Caller{Scope: "5.6.7.8", Account: "someone-else"}))
	assert.Equal(t, 2, bans.Flagged(), "one guest, banned twice")
}

func TestAFlaggedSignedInAccountLeavesItsScopeAlone(t *testing.T) {
	bans, _, _ := newBans(t, newClock())

	_, accepted := bans.Flag(shadowban.Caller{Scope: "campus", Account: "player", SignedIn: true})
	require.True(t, accepted)

	assert.True(t, bans.Banned(shadowban.Caller{Scope: "home", Account: "player", SignedIn: true}))
	assert.False(t, bans.Banned(shadowban.Caller{Scope: "campus", Account: "classmate", SignedIn: true}),
		"the other players behind the campus address are not banned with it")
}

func TestAFlagWithNoAccountBansTheScope(t *testing.T) {
	bans, _, _ := newBans(t, newClock())

	_, accepted := bans.Flag(shadowban.Caller{Scope: "1.2.3.4"})
	require.True(t, accepted)

	assert.True(t, bans.Banned(shadowban.Caller{Scope: "1.2.3.4"}))
	assert.True(t, bans.Banned(shadowban.Caller{Scope: "1.2.3.4", Account: "guest"}))
}

func TestAnOperatorBanOnAnAccountBansTheAccountAlone(t *testing.T) {
	clock := newClock()
	bans, _, _ := newBans(t, clock)

	sentence := bans.Ban(shadowban.Caller{Account: "guest"}, 0)
	assert.Equal(t, 1, sentence.Offence)
	assert.Equal(t, clock.Now().Add(time.Hour), sentence.Until)

	assert.True(t, bans.Banned(shadowban.Caller{Scope: "1.2.3.4", Account: "guest"}))
	assert.False(t, bans.Banned(shadowban.Caller{Scope: "1.2.3.4"}))
}

func TestTheSentenceIsTheBanThatEndsLast(t *testing.T) {
	clock := newClock()
	bans, _, _ := newBans(t, clock)

	bans.Ban(shadowban.Caller{Scope: "1.2.3.4"}, time.Hour)
	bans.Ban(shadowban.Caller{Account: "guest"}, 2*time.Hour)

	sentence, running := bans.Sentence(shadowban.Caller{Scope: "1.2.3.4", Account: "guest"})
	require.True(t, running)
	assert.Equal(t, clock.Now().Add(2*time.Hour), sentence.Until)

	_, running = bans.Sentence(shadowban.Caller{Scope: "5.6.7.8", Account: "someone-else"})
	assert.False(t, running)
}

func TestAccountBansAreKeptApartAndSurviveARestart(t *testing.T) {
	clock := newClock()
	before, scopes, accounts := newBans(t, clock)

	before.Flag(shadowban.Caller{Scope: "1.2.3.4", Account: "guest"})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	before.Run(ctx)

	assert.Contains(t, scopes.Stored(), "1.2.3.4")
	assert.NotContains(t, scopes.Stored(), "guest")
	assert.Contains(t, accounts.Stored(), "guest")

	after := shadowban.NewBans(config(), clock, scopes, accounts, failOnStateError(t))
	require.NoError(t, after.Load(t.Context()))
	assert.True(t, after.Banned(shadowban.Caller{Scope: "5.6.7.8", Account: "guest"}))
}
