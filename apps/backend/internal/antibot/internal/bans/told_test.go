package bans_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func toldBook(t *testing.T, config bans.Config, clock cptime.Clock, scopes, accounts bans.Store) (book, *bans.MemoryEnforcer) {
	t.Helper()
	enforcer := &bans.MemoryEnforcer{}
	return book{t: t, inner: bans.NewBook(config, pardon, clock, scopes, accounts, cpatomic.FakeTransactioner{}, enforcer)}, enforcer
}

func TestAFlagTellsTheGameToRefuseTheGuestsAccountAndScopeUntilTheBanEnds(t *testing.T) {
	clock := newClock()
	book, told := toldBook(t, config(), clock, bans.NewMemoryStore(), bans.NewMemoryStore())

	sentence, accepted := book.Flag(bans.Caller{Scope: "1.2.3.4", Account: "guest"})
	require.True(t, accepted)

	assert.Equal(t, []bans.Told{
		{Caller: bans.Caller{Account: "guest"}, Until: sentence.Until, At: clock.Now()},
		{Caller: bans.Caller{Scope: "1.2.3.4"}, Until: sentence.Until, At: clock.Now()},
	}, told.Told())
}

func TestAFlagThatChangesNothingTellsNothing(t *testing.T) {
	clock := newClock()
	book, told := toldBook(t, config(), clock, bans.NewMemoryStore(), bans.NewMemoryStore())
	player := bans.Caller{Account: "player", SignedIn: true}

	book.Flag(player)
	clock.Advance(time.Minute)
	_, accepted := book.Flag(player)

	require.False(t, accepted, "inside the reflag interval")
	assert.Len(t, told.Told(), 1)
}

func TestAnUnbanTellsTheGameToAdmitTheCallerNow(t *testing.T) {
	clock := newClock()
	book, told := toldBook(t, config(), clock, bans.NewMemoryStore(), bans.NewMemoryStore())
	book.Ban(bans.Caller{Scope: "1.2.3.4"}, time.Hour)

	clock.Advance(time.Minute)
	book.Unban(bans.Caller{Scope: "1.2.3.4"})

	require.Len(t, told.Told(), 2)
	assert.Equal(t, bans.Told{Caller: bans.Caller{Scope: "1.2.3.4"}, Until: clock.Now(), At: clock.Now()}, told.Told()[1])
}

func TestABanThatIsNotEnforcedTellsTheGameToRefuseNothing(t *testing.T) {
	clock := newClock()
	watching := config()
	watching.Enforce = false
	book, told := toldBook(t, watching, clock, bans.NewMemoryStore(), bans.NewMemoryStore())

	book.Ban(bans.Caller{Account: "player", SignedIn: true}, time.Hour)

	assert.Equal(t, []bans.Told{{Caller: bans.Caller{Account: "player"}, Until: clock.Now(), At: clock.Now()}}, told.Told())
}

func TestABanTheStoreRefusedIsNeverTold(t *testing.T) {
	scopes := bans.NewMemoryStore()
	scopes.FailWith(errors.New("postgres is down"))
	enforcer := &bans.MemoryEnforcer{}
	inner := bans.NewBook(config(), pardon, newClock(), scopes, bans.NewMemoryStore(), cpatomic.FakeTransactioner{}, enforcer)

	_, err := inner.Ban(t.Context(), bans.Caller{Scope: "1.2.3.4"}, time.Hour)

	require.Error(t, err)
	assert.Empty(t, enforcer.Told())
}

func TestEveryRunningBanIsToldAgain(t *testing.T) {
	clock := newClock()
	scopes := bans.NewMemoryStore(
		bans.Record{Key: "1.2.3.4", Offences: 1, ExpiresAt: clock.Now().Add(time.Hour)},
		bans.Record{Key: "5.6.7.8", Offences: 1, ExpiresAt: clock.Now().Add(-time.Hour)},
	)
	accounts := bans.NewMemoryStore(bans.Record{Key: "guest", Offences: 2, ExpiresAt: clock.Now().Add(24 * time.Hour)})
	book, told := toldBook(t, config(), clock, scopes, accounts)

	require.NoError(t, book.inner.Restate(t.Context()))

	assert.Equal(t, []bans.Told{
		{Caller: bans.Caller{Scope: "1.2.3.4"}, Until: clock.Now().Add(time.Hour), At: clock.Now()},
		{Caller: bans.Caller{Account: "guest"}, Until: clock.Now().Add(24 * time.Hour), At: clock.Now()},
	}, told.Told(), "a ban that ended is not told: the game refuses it no longer either way")
}
