package antibot_test

import (
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
)

func (s *stack) shed(r *rand.Rand, scope, account, flag string) (int, bool) {
	for i := range 62 {
		s.clock.Advance(time.Duration(150+r.IntN(300)) * time.Millisecond)
		if s.clickAs(scope, account, 100000+uint32(r.IntN(60000)), flag) {
			return i, true
		}
	}
	return 0, false
}

func TestTheHomeLineThatShedsItsAccountIsCaught(t *testing.T) {
	s := newStack()

	//nolint:gosec // seeded test PRNG
	r := rand.New(rand.NewPCG(30, 9))

	for i := range 10 {
		at, dropped := s.shed(r, "2001:db8:a:1::/64", fmt.Sprintf("guest-%d", i), "pl")
		s.clock.Advance(time.Duration(30+r.IntN(30)) * time.Second)

		if i < 5 {
			assert.False(t, dropped, "account %d: a household could hold that many", i)
			continue
		}
		require.True(t, dropped, "account %d", i)
		assert.Zero(t, at, "account %d is dropped from its first click", i)
	}

	verdicts := s.verdicts("2001:db8:a:1::/64")
	assert.Equal(t, detect.Certain, verdicts["churner"])
	for _, watchdog := range []string{"retaker", "sequencer", "metronome", "catcher", "cohort", "scraper"} {
		assert.Equal(t, detect.Clear, verdicts[watchdog], "%s: each account lived a minute", watchdog)
	}
}

func TestTheRelayOverFreshMobileLinesIsCaught(t *testing.T) {
	s := newStack()

	//nolint:gosec // seeded test PRNG
	r := rand.New(rand.NewPCG(30, 10))

	for i := range 10 {
		scope := fmt.Sprintf("2001:db8:%x:%x::/64", r.IntN(0x2300), r.IntN(0xffff))
		at, dropped := s.shed(r, scope, fmt.Sprintf("guest-%d", i), "dz")
		s.clock.Advance(time.Duration(30+r.IntN(40)) * time.Second)

		if i < 6 {
			assert.False(t, dropped, "line %d", i)
			continue
		}
		require.True(t, dropped, "line %d", i)
		assert.LessOrEqual(t, at, 20, "line %d keeps less than a third of its bank", i)
		assert.Equal(t, detect.Certain, s.verdicts(scope)["churner"])
	}
}

func TestAFamilyOnOneLineIsNotBanned(t *testing.T) {
	s := newStack()

	//nolint:gosec // seeded test PRNG
	r := rand.New(rand.NewPCG(30, 11))

	for range 120 {
		for _, account := range []string{"parent", "child", "other-child"} {
			s.clock.Advance(time.Duration(500+r.IntN(4000)) * time.Millisecond)
			require.False(t, s.clickAs("2001:db8:1:2::/64", account, 100000+uint32(r.IntN(60000)), "fr"))
		}
	}

	assert.Empty(t, s.reports)
}
