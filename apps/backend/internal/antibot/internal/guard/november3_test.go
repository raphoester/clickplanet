package guard_test

import (
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/warnings"
)

const (
	russianAddress = "192.0.2.61"
	brazilianLine  = "2001:db8:2c44:1201::/64"
)

func vpnExit(i int) string { return fmt.Sprintf("2001:db8:bac1:%x::/64", i) }

// One hand, two tabs: each spends its bank in a burst, and the other takes over a few seconds later. It stops at a ban
// or a warning.
func (s *stack) switchTabs(r *rand.Rand, lasting time.Duration, tabs [2]func() (scope, account string), switchAfter, idle time.Duration) {
	told := len(s.reports) + len(s.warnings)
	for start := s.clock.Now(); s.clock.Now().Sub(start) < lasting && len(s.reports)+len(s.warnings) == told; {
		for _, tab := range tabs {
			scope, account := tab()
			s.burst(r, scope, account, "RU", 16*time.Second, 900*time.Millisecond)
			s.rest(s.clock.Now().Add(switchAfter/2 + time.Duration(r.Int64N(int64(switchAfter)))))
		}
		s.rest(s.clock.Now().Add(idle))
	}
}

func accountsOf(reports []guard.Report) []string {
	accounts := make([]string, 0, len(reports))
	for _, report := range reports {
		accounts = append(accounts, report.Account)
	}
	return accounts
}

func (s *stack) reported() []string { return accountsOf(s.reports) }

func (s *stack) warnedAccounts() []string { return accountsOf(s.warnings) }

func TestTheAccountOfNovember3AndItsGuestOnAVPNAreWarnedThenBanned(t *testing.T) {
	s := newStack(t)
	s.signIn("bonapart")

	//nolint:gosec // seeded test PRNG
	r := rand.New(rand.NewPCG(11, 3))

	exit := 0
	tabs := [2]func() (string, string){
		func() (string, string) { return russianAddress, "bonapart" },
		func() (string, string) {
			if r.IntN(10) == 0 {
				exit++
			}
			return vpnExit(exit), "vpn-guest"
		},
	}

	s.switchTabs(r, 3*time.Hour+45*time.Minute, tabs, 8*time.Second, 50*time.Second)
	require.Empty(t, s.warnings, "under four hours of turns")
	require.Greater(t, exit, 10, "the guest clicked from many exits")

	s.switchTabs(r, 30*time.Minute, tabs, 8*time.Second, 50*time.Second)

	assert.ElementsMatch(t, []string{"bonapart", "vpn-guest"}, s.warnedAccounts())
	assert.Empty(t, s.reports, "a ring is warned before it is banned")
	assert.Contains(t, s.rises, "rotator certain")
	for _, rise := range s.rises {
		assert.Regexp(t, `^rotator `, rise, "no other watchdog read anything, on any exit")
	}

	s.switchTabs(r, 110*time.Minute, tabs, 8*time.Second, 50*time.Second)
	require.Empty(t, s.reports, "under two hours of turns since the warning")

	s.switchTabs(r, 30*time.Minute, tabs, 8*time.Second, 50*time.Second)

	assert.ElementsMatch(t, []string{"bonapart", "vpn-guest"}, s.reported())

	assert.Empty(t, s.bans.Stored(), "turns name accounts: no address and no exit is banned")
	assert.False(t, s.clickAs(vpnExit(exit), "stranger", 1, "RU"), "a stranger on the same exit plays")
	assert.False(t, s.clickAs(russianAddress, "fresh-cookie", 1, "RU"))
	assert.True(t, s.clickAs(vpnExit(exit+1), "vpn-guest", 1, "RU"), "the guest is banned on any exit")
}

func TestThePairOfNovember3OnOneAddressIsWarnedThenBanned(t *testing.T) {
	s := newStack(t)
	s.signIn("lynx", "zebra")

	//nolint:gosec // seeded test PRNG
	r := rand.New(rand.NewPCG(11, 4))

	tabs := [2]func() (string, string){
		func() (string, string) { return russianAddress, "lynx" },
		func() (string, string) { return russianAddress, "zebra" },
	}

	s.switchTabs(r, 3*time.Hour+45*time.Minute, tabs, 4*time.Second, 10*time.Second)
	require.Empty(t, s.warnings)

	s.switchTabs(r, 30*time.Minute, tabs, 4*time.Second, 10*time.Second)
	require.ElementsMatch(t, []string{"lynx", "zebra"}, s.warnedAccounts())
	warnedAt := s.teller.Told()[0].At

	s.switchTabs(r, 3*time.Hour, tabs, 4*time.Second, 10*time.Second)

	assert.ElementsMatch(t, []string{"lynx", "zebra"}, s.reported())
	assert.WithinRange(t, s.clock.Now(), warnedAt.Add(115*time.Minute), warnedAt.Add(130*time.Minute),
		"24 slots of turns after the slot of the warning")
	examination := s.examine(russianAddress, "lynx")
	assert.Regexp(t, `^rotation accounts=2 byChance=\d+ flag=RU sharedSeconds=0 sharedSlots=24$`, examination.Readings[9].Evidence)
	assert.Equal(t, warnedAt, examination.WarnedAt)
	assert.Empty(t, s.bans.Stored(), "two signed-in accounts leave their address alone")
}

func TestARingThatStopsAtItsWarningPlaysOn(t *testing.T) {
	s := newStack(t)
	s.signIn("lynx", "zebra")

	//nolint:gosec // seeded test PRNG
	r := rand.New(rand.NewPCG(11, 4))

	s.switchTabs(r, 5*time.Hour, [2]func() (string, string){
		func() (string, string) { return russianAddress, "lynx" },
		func() (string, string) { return russianAddress, "zebra" },
	}, 4*time.Second, 10*time.Second)
	require.ElementsMatch(t, []string{"lynx", "zebra"}, s.warnedAccounts())

	for until := s.clock.Now().Add(3 * time.Hour); s.clock.Now().Before(until); {
		require.False(t, s.burst(r, russianAddress, "lynx", "RU", 16*time.Second, 900*time.Millisecond))
		s.rest(s.clock.Now().Add(20 * time.Second))
	}

	assert.Empty(t, s.reports, "the turns that earned the warning never ban")
}

func TestABootForgetsTheTurnsBeforeAWarningTheEvidenceStillHeld(t *testing.T) {
	s := newStack(t)
	s.signIn("lynx", "zebra")
	s.set.Bans.Enforce = proto.Bool(false)
	s.play(s.set)

	//nolint:gosec // seeded test PRNG
	r := rand.New(rand.NewPCG(11, 4))
	tabs := [2]func() (string, string){
		func() (string, string) { return russianAddress, "lynx" },
		func() (string, string) { return russianAddress, "zebra" },
	}

	s.switchTabs(r, 5*time.Hour, tabs, 4*time.Second, 10*time.Second)
	require.Empty(t, s.warnings, "not enforced, no ring is warned")
	_, err := s.warned.Keep(t.Context(), warnings.Warning{Account: "lynx", At: s.clock.Now()})
	require.NoError(t, err)

	s.set.Bans.Enforce = proto.Bool(true)
	s.play(s.set)
	s.restart(time.Minute)
	s.switchTabs(r, 10*time.Minute, tabs, 4*time.Second, 10*time.Second)

	assert.Empty(t, s.reports, "the evidence saved before the warning was forgotten at the boot")
}

func TestAShortBurstOfTurnsBansNobody(t *testing.T) {
	s := newStack(t)
	s.signIn("lemur")

	//nolint:gosec // seeded test PRNG
	r := rand.New(rand.NewPCG(11, 5))

	s.switchTabs(r, 35*time.Minute, [2]func() (string, string){
		func() (string, string) { return brazilianLine, "lemur" },
		func() (string, string) { return brazilianLine, "koala" },
	}, 6*time.Second, 30*time.Second)

	assert.Empty(t, s.reports)
	assert.Empty(t, s.rises, "seven slots are too few to read")
}
