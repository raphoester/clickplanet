package metronome_test

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/metronome"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const scope = "2001:db8:1:2::/64"

type spenders struct {
	watchdog *metronome.Watchdog
	clock    *cptime.FixedClock
}

func newSpenders(config metronome.Config) *spenders {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 28, 21, 51, 0, 0, time.UTC))
	return &spenders{watchdog: metronome.New(config, clock, func(float64) {}, func(time.Duration) {}), clock: clock}
}

func (s *spenders) click(account string) (detect.Verdict, detect.Evidence) {
	click := detect.Click{Scope: scope, Account: account, Tile: 1, Country: "DZ", At: s.clock.Now()}
	s.watchdog.Attempted(click)
	return s.watchdog.Watch(click)
}

func (s *spenders) every(gap, d time.Duration, account string) (detect.Verdict, detect.Evidence) {
	var (
		verdict  detect.Verdict
		evidence detect.Evidence
	)
	for end := s.clock.Now().Add(d); s.clock.Now().Before(end); {
		s.clock.Advance(gap)
		verdict, evidence = s.click(account)
	}
	return verdict, evidence
}

// nightBot is the bot of 2026-09-28: the refill of a guest's allowance, and five minutes off every half hour.
func (s *spenders) nightBot(d time.Duration, account string) (detect.Verdict, detect.Evidence) {
	var (
		verdict  detect.Verdict
		evidence detect.Evidence
	)
	for end := s.clock.Now().Add(d); s.clock.Now().Before(end); {
		verdict, evidence = s.every(5*time.Second, 25*time.Minute, account)
		s.clock.Advance(5 * time.Minute)
	}
	return verdict, evidence
}

func staminaConfig() metronome.Config {
	return metronome.Config{
		TrackWindow: 15 * time.Minute,
		Stamina: metronome.StaminaConfig{
			Slice:       10 * time.Minute,
			Clicks:      40,
			Window:      6 * time.Hour,
			MinBusy:     4 * time.Hour,
			CertainBusy: 5*time.Hour + 30*time.Minute,
		},
	}
}

func TestHoursAtThePaceOfTheAllowanceAreCertain(t *testing.T) {
	s := newSpenders(staminaConfig())

	verdict, evidence := s.nightBot(6*time.Hour, "night")

	require.Equal(t, detect.Certain, verdict)
	assert.Equal(t, "stamina", evidence.Rule)
	assert.Contains(t, evidence.Fields, detect.Field{Key: "window", Value: 6 * time.Hour})
}

func TestFourHoursAreOnlySuspect(t *testing.T) {
	s := newSpenders(staminaConfig())

	verdict, _ := s.nightBot(4*time.Hour+10*time.Minute, "night")

	assert.Equal(t, detect.Suspect, verdict)
}

func TestAPlayerWhoStopsIsNeverCertain(t *testing.T) {
	s := newSpenders(staminaConfig())
	r := rand.New(rand.NewPCG(20, 9)) //nolint:gosec // a test draw, not a secret.

	for s.clock.Now().Before(time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)) {
		for end := s.clock.Now().Add(time.Duration(20+r.IntN(30)) * time.Minute); s.clock.Now().Before(end); {
			s.clock.Advance(time.Duration(1500+r.IntN(5000)) * time.Millisecond)
			verdict, evidence := s.click("player")
			require.NotEqual(t, detect.Certain, verdict, evidence.String())
		}
		s.clock.Advance(time.Duration(15+r.IntN(45)) * time.Minute)
	}
}

func TestClickingSlowlyAllNightIsNotSpending(t *testing.T) {
	s := newSpenders(staminaConfig())

	verdict, _ := s.every(20*time.Second, 8*time.Hour, "night")

	assert.Equal(t, detect.Clear, verdict, "thirty clicks a slice is under the bound")
}

func TestAccountsTakingTurnsOnOneScopeAreEachJudgedAlone(t *testing.T) {
	s := newSpenders(staminaConfig())

	var verdict detect.Verdict
	for _, account := range []string{"a", "b", "c", "d", "e", "f", "g"} {
		verdict, _ = s.nightBot(time.Hour, account)
		require.Equal(t, detect.Clear, verdict)
	}
}

func TestAClickWithNoAccountSpendsItsScopes(t *testing.T) {
	s := newSpenders(staminaConfig())

	verdict, evidence := s.nightBot(6*time.Hour, "")

	require.Equal(t, detect.Certain, verdict)
	assert.Equal(t, "stamina", evidence.Rule)
}

func TestOnlyTheWindowCounts(t *testing.T) {
	s := newSpenders(staminaConfig())

	_, _ = s.nightBot(6*time.Hour, "night")
	s.clock.Advance(3 * time.Hour)

	verdict, _ := s.click("night")

	assert.Equal(t, detect.Clear, verdict)
}

func TestStaminaWithNoBoundOnlyMeasures(t *testing.T) {
	c := staminaConfig()
	c.Stamina.MinBusy = 0
	c.Stamina.CertainBusy = 0
	s := newSpenders(c)

	verdict, _ := s.nightBot(6*time.Hour, "night")

	assert.Equal(t, detect.Clear, verdict)
}

func TestAThrottledTryIsNotSpent(t *testing.T) {
	s := newSpenders(staminaConfig())

	for end := s.clock.Now().Add(6 * time.Hour); s.clock.Now().Before(end); {
		s.clock.Advance(5 * time.Second)
		s.watchdog.Attempted(detect.Click{Scope: scope, Account: "night", At: s.clock.Now()})
	}

	verdict, _ := s.click("night")

	assert.Equal(t, detect.Clear, verdict)
}
