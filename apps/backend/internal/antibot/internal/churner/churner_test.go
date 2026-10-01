package churner_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/churner"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// The bounds cmd/api/example.yaml ships.
func config() churner.Config {
	return churner.Config{
		Window: time.Hour,
		V6:     churner.Bounds{MinAccounts: 4, CertainAccounts: 6},
		V4:     churner.Bounds{MinAccounts: 10, CertainAccounts: 20},
		Relay: churner.Relay{
			Handoff:      90 * time.Second,
			MaxLife:      5 * time.Minute,
			MinClicks:    20,
			MinFlagShare: 0.9,
			V4Bits:       24,
			V6Bits:       32,
			MinLinks:     3,
			CertainLinks: 6,
		},
	}
}

func churnOnly() churner.Config {
	c := config()
	c.Relay.MinLinks, c.Relay.CertainLinks = 0, 0
	return c
}

type harness struct {
	watchdog *churner.Watchdog
	clock    *cptime.FixedClock
}

func newHarness(config churner.Config) *harness {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 30, 13, 47, 0, 0, time.UTC))
	return &harness{
		watchdog: churner.New(config, clock, func(int, string) {}, func(int) {}),
		clock:    clock,
	}
}

type reading struct {
	verdict  detect.Verdict
	evidence detect.Evidence
}

// spend is one guest account clicking clicks times, gap apart, and its strongest reading.
func (h *harness) spend(scope, account, flag string, clicks int, gap time.Duration) reading {
	var strongest reading
	for range clicks {
		h.clock.Advance(gap)
		verdict, evidence := h.watchdog.Watch(detect.Click{Scope: scope, Account: account, Tile: 1, Country: flag, At: h.clock.Now()})
		if verdict >= strongest.verdict {
			strongest = reading{verdict: verdict, evidence: evidence}
		}
	}
	return strongest
}

func TestAFreshAccountEveryMinuteOnOneLineIsCaught(t *testing.T) {
	h := newHarness(config())

	readings := make([]reading, 0, 8)
	for i := range 8 {
		readings = append(readings, h.spend("2001:db8:a:1::/64", fmt.Sprintf("guest-%d", i), "pl", 62, 300*time.Millisecond))
		h.clock.Advance(40 * time.Second)
	}

	for i, r := range readings[:3] {
		assert.Equal(t, detect.Clear, r.verdict, "account %d", i)
	}
	assert.Equal(t, detect.Suspect, readings[3].verdict)
	for i, r := range readings[5:] {
		require.Equal(t, detect.Certain, r.verdict, "account %d", i+5)
		assert.Equal(t, "churn", r.evidence.Rule)
	}
	assert.Contains(t, readings[5].evidence.Fields, detect.Field{Key: "accounts", Value: 6})
}

func TestAHouseholdKeepsItsAccounts(t *testing.T) {
	h := newHarness(config())

	var strongest detect.Verdict
	for range 60 {
		for _, account := range []string{"parent", "child", "other-child"} {
			r := h.spend("2001:db8:1:2::/64", account, "fr", 10, 2*time.Second)
			strongest = max(strongest, r.verdict)
		}
	}

	assert.Equal(t, detect.Clear, strongest)
}

func TestAnIPv4AddressIsHeldToItsOwnBounds(t *testing.T) {
	for scope, want := range map[string]detect.Verdict{
		"2001:db8:a:1::/64": detect.Certain,
		"203.0.113.7":       detect.Clear,
	} {
		h := newHarness(churnOnly())

		var strongest detect.Verdict
		for i := range 8 {
			r := h.spend(scope, fmt.Sprintf("guest-%d", i), "pl", 30, time.Second)
			strongest = max(strongest, r.verdict)
		}

		assert.Equal(t, want, strongest, scope)
	}
}

func TestAnIPv4AddressStillReadsPastItsBounds(t *testing.T) {
	h := newHarness(churnOnly())

	var last reading
	for i := range 20 {
		last = h.spend("203.0.113.7", fmt.Sprintf("guest-%d", i), "pl", 30, time.Second)
	}

	assert.Equal(t, detect.Certain, last.verdict)
	assert.Contains(t, last.evidence.Fields, detect.Field{Key: "family", Value: "v4"})
}

func TestASignedInAccountIsNeverCounted(t *testing.T) {
	h := newHarness(config())

	for i := range 8 {
		h.spend("2001:db8:a:1::/64", fmt.Sprintf("guest-%d", i), "pl", 30, time.Second)
	}

	verdict, _ := h.watchdog.Watch(detect.Click{Scope: "2001:db8:a:1::/64", Account: "linked", SignedIn: true, Tile: 1, Country: "pl", At: h.clock.Now()})
	assert.Equal(t, detect.Clear, verdict)
}

func TestAccountsBornAnHourAgoNoLongerCount(t *testing.T) {
	h := newHarness(config())

	for i := range 8 {
		h.spend("2001:db8:a:1::/64", fmt.Sprintf("guest-%d", i), "pl", 30, time.Second)
	}
	h.clock.Advance(time.Hour)

	assert.Equal(t, detect.Clear, h.spend("2001:db8:a:1::/64", "guest-7", "pl", 1, time.Second).verdict)
}

func relayIdentity(i int) (scope, account string) {
	return fmt.Sprintf("2001:db8:%x:%x::/64", 0x100+i*0x37, i), fmt.Sprintf("guest-%d", i)
}

func TestAFreshAccountOnAFreshLineEveryMinuteIsARelay(t *testing.T) {
	h := newHarness(config())

	readings := make([]reading, 0, 10)
	for i := range 10 {
		scope, account := relayIdentity(i)
		readings = append(readings, h.spend(scope, account, "dz", 62, 300*time.Millisecond))
		h.clock.Advance(45 * time.Second)
	}

	assert.Equal(t, detect.Clear, readings[0].verdict, "the first account took over from nobody")
	assert.Equal(t, detect.Suspect, readings[3].verdict)
	for i, r := range readings[6:] {
		require.Equal(t, detect.Certain, r.verdict, "account %d", i+6)
		assert.Equal(t, "relay", r.evidence.Rule)
		assert.Contains(t, r.evidence.Fields, detect.Field{Key: "prefix", Value: "2001:db8::/32"})
	}
}

func TestAPlayerWhoseLineChangesKeepsTheirAccount(t *testing.T) {
	h := newHarness(config())

	var strongest detect.Verdict
	for i := range 20 {
		scope, _ := relayIdentity(i)
		strongest = max(strongest, h.spend(scope, "the-player", "dz", 62, 300*time.Millisecond).verdict)
		h.clock.Advance(45 * time.Second)
	}

	assert.Equal(t, detect.Clear, strongest)
}

func TestPlayersWhoStayAreNoRelay(t *testing.T) {
	h := newHarness(config())

	var strongest detect.Verdict
	for i := range 10 {
		scope, account := relayIdentity(i)
		strongest = max(strongest, h.spend(scope, account, "dz", 200, 2*time.Second).verdict)
		h.clock.Advance(45 * time.Second)
	}

	assert.Equal(t, detect.Clear, strongest, "each one played past maxLife before the next one came")
}

func TestAccountsOnOtherFlagsAreNoRelay(t *testing.T) {
	h := newHarness(config())

	var strongest detect.Verdict
	for i := range 10 {
		scope, account := relayIdentity(i)
		flag := []string{"dz", "pl"}[i%2]
		strongest = max(strongest, h.spend(scope, account, flag, 62, 300*time.Millisecond).verdict)
		h.clock.Advance(45 * time.Second)
	}

	assert.Equal(t, detect.Clear, strongest)
}

func TestUnsetBoundsNeverRead(t *testing.T) {
	h := newHarness(churner.Config{})

	var strongest detect.Verdict
	for i := range 10 {
		strongest = max(strongest, h.spend("2001:db8:a:1::/64", fmt.Sprintf("guest-%d", i), "pl", 62, 300*time.Millisecond).verdict)
		scope, account := relayIdentity(i)
		strongest = max(strongest, h.spend(scope, account, "dz", 62, 300*time.Millisecond).verdict)
		h.clock.Advance(45 * time.Second)
	}

	assert.Equal(t, detect.Clear, strongest)
}
