package guard_test

import (
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	russianAddress = "192.0.2.61"
	brazilianLine  = "2001:db8:2c44:1201::/64"
)

func vpnExit(i int) string { return fmt.Sprintf("2001:db8:bac1:%x::/64", i) }

// One hand, two tabs: each spends its bank in a burst, and the other takes over a few seconds later. It stops at a ban.
func (s *stack) switchTabs(r *rand.Rand, lasting time.Duration, tabs [2]func() (scope, account string), switchAfter, idle time.Duration) {
	for start := s.clock.Now(); s.clock.Now().Sub(start) < lasting && len(s.reports) == 0; {
		for _, tab := range tabs {
			scope, account := tab()
			s.burst(r, scope, account, "RU", 16*time.Second, 900*time.Millisecond)
			s.rest(s.clock.Now().Add(switchAfter/2 + time.Duration(r.Int64N(int64(switchAfter)))))
		}
		s.rest(s.clock.Now().Add(idle))
	}
}

func (s *stack) reported() []string {
	accounts := make([]string, 0, len(s.reports))
	for _, report := range s.reports {
		accounts = append(accounts, report.Account)
	}
	return accounts
}

func TestTheAccountOfNovember3AndItsGuestOnAVPNAreBanned(t *testing.T) {
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
	require.Empty(t, s.reports, "under four hours of turns")
	require.Greater(t, exit, 10, "the guest clicked from many exits")

	s.switchTabs(r, 30*time.Minute, tabs, 8*time.Second, 50*time.Second)

	assert.ElementsMatch(t, []string{"bonapart", "vpn-guest"}, s.reported())
	assert.Contains(t, s.rises, "rotator certain")
	for _, rise := range s.rises {
		assert.Regexp(t, `^rotator `, rise, "no other watchdog read anything, on any exit")
	}

	assert.Empty(t, s.bans.Stored(), "turns name accounts: no address and no exit is banned")
	assert.False(t, s.clickAs(vpnExit(exit), "stranger", 1, "RU"), "a stranger on the same exit plays")
	assert.False(t, s.clickAs(russianAddress, "fresh-cookie", 1, "RU"))
	assert.True(t, s.clickAs(vpnExit(exit+1), "vpn-guest", 1, "RU"), "the guest is banned on any exit")
}

func TestThePairOfNovember3OnOneAddressIsBanned(t *testing.T) {
	s := newStack(t)
	s.signIn("lynx", "zebra")

	//nolint:gosec // seeded test PRNG
	r := rand.New(rand.NewPCG(11, 4))

	tabs := [2]func() (string, string){
		func() (string, string) { return russianAddress, "lynx" },
		func() (string, string) { return russianAddress, "zebra" },
	}

	s.switchTabs(r, 3*time.Hour+45*time.Minute, tabs, 4*time.Second, 10*time.Second)
	require.Empty(t, s.reports)

	s.switchTabs(r, 30*time.Minute, tabs, 4*time.Second, 10*time.Second)

	assert.ElementsMatch(t, []string{"lynx", "zebra"}, s.reported())
	examination := s.examine(russianAddress, "lynx")
	assert.Regexp(t, `^rotation accounts=2 byChance=\d+ flag=RU sharedSeconds=0 sharedSlots=4\d$`, examination.Readings[9].Evidence)
	assert.Empty(t, s.bans.Stored(), "two signed-in accounts leave their address alone")
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
