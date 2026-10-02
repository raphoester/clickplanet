package churner

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func bounded() Config {
	return Config{
		V6:    Bounds{MinAccounts: 4, CertainAccounts: 6},
		Relay: Relay{MinLinks: 3, CertainLinks: 6},
	}
}

func newWatchdog(config Config, clock cptime.Clock) *Watchdog {
	return New(config, clock, func(int, string) {}, func(int) {})
}

func churn(w *Watchdog, clock *cptime.FixedClock, accounts int) {
	for i := range accounts {
		scope := fmt.Sprintf("2001:db8:%x::/64", 0x100+i)
		for range 30 {
			clock.Advance(500 * time.Millisecond)
			w.Watch(detect.Click{Scope: scope, Account: fmt.Sprintf("guest-%d", i), Tile: 1, Country: "dz", At: clock.Now()})
			w.Watch(detect.Click{Scope: "2001:db8:a:1::/64", Account: fmt.Sprintf("home-%d", i), Tile: 1, Country: "pl", At: clock.Now()})
		}
		clock.Advance(30 * time.Second)
	}
}

func TestAccountsSurviveASaveAndLoad(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 30, 13, 47, 0, 0, time.UTC))

	w := newWatchdog(bounded(), clock)
	churn(w, clock, 7)

	data, err := w.Save()
	require.NoError(t, err)

	restarted := newWatchdog(bounded(), clock)
	require.NoError(t, restarted.Load(data))

	clock.Advance(30 * time.Second)
	for _, next := range []detect.Click{
		{Scope: "2001:db8:a:1::/64", Account: "home-7", Tile: 1, Country: "pl", At: clock.Now()},
		{Scope: "2001:db8:106::/64", Account: "guest-6", Tile: 1, Country: "dz", At: clock.Now()},
	} {
		wantVerdict, wantEvidence := w.Watch(next)
		gotVerdict, gotEvidence := restarted.Watch(next)

		require.Equal(t, detect.Certain, wantVerdict, next.Scope)
		assert.Equal(t, wantVerdict, gotVerdict, next.Scope)
		assert.Equal(t, wantEvidence, gotEvidence, next.Scope)
	}
	assert.Equal(t, w.prefixes, restarted.prefixes)
}

func TestForgetDropsAccountsAndTheirIndexes(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 30, 13, 47, 0, 0, time.UTC))

	w := newWatchdog(bounded(), clock)
	churn(w, clock, 2)

	w.Forget(clock.Now().Add(time.Second))

	assert.Empty(t, w.accounts)
	assert.Empty(t, w.born)
	assert.Empty(t, w.prefixes)
}
