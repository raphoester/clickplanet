package antibot_test

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const polishLine = "polish-line"

func (s *stack) takeTurns(r *rand.Rand, lasting time.Duration) {
	for start := s.clock.Now(); s.clock.Now().Sub(start) < lasting; {
		cycle := s.clock.Now()

		require.False(s.t, s.burst(r, polishLine, "desktop", "PL", 25*time.Second, time.Second))
		s.rest(cycle.Add(time.Minute))
		require.False(s.t, s.burst(r, polishLine, "laptop", "PL", 25*time.Second, time.Second))
		s.rest(cycle.Add(150 * time.Second))
	}
}

func TestThePairOfOctober13TakingTurnsAtACadenceIsBanned(t *testing.T) {
	s := newStack(t, withTheBoundsOfOctober9, productionRetakes)
	s.signIn("desktop", "laptop")

	//nolint:gosec // seeded test PRNG
	r := rand.New(rand.NewPCG(10, 13))

	s.takeTurns(r, 2*time.Hour+30*time.Minute)
	require.Equal(t, []string{"rotator suspect"}, s.rises, "taking turns alone bans nobody")

	var dropped bool
	for range 200 {
		s.clock.Advance(time.Duration(200+r.IntN(380)) * time.Millisecond)
		if s.clickAs(polishLine, "laptop", 100000+uint32(r.IntN(60000)), "PL") {
			dropped = true
			break
		}
	}

	require.True(t, dropped, "a cadence beside the turns is two watchdogs")
	require.NotEmpty(t, s.reports)

	report := s.reports[0]
	assert.Equal(t, "laptop", report.Account)
	assert.Regexp(t, `^suspect cadence clicks=12\d median=3\d\d\.?\d*ms spread=\S+ sustained=\S+$`, report.Opinions[2].String())
	assert.Regexp(t, `^suspect rotation accounts=2 `, report.Opinions[9].String())

	banned := map[string]string{}
	for _, report := range s.reports {
		banned[report.Account] = report.Alongside
	}
	assert.Equal(t, map[string]string{"laptop": "", "desktop": "laptop"}, banned, "the ring is banned together")
}
