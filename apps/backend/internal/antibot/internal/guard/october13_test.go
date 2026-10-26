package guard_test

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const polishLine = "polish-line"

func (s *stack) takeTurns(r *rand.Rand, scope, flag string, lasting time.Duration, first, second string) {
	for start := s.clock.Now(); s.clock.Now().Sub(start) < lasting; {
		cycle := s.clock.Now()

		require.False(s.t, s.burst(r, scope, first, flag, 25*time.Second, time.Second))
		s.rest(cycle.Add(time.Minute))
		require.False(s.t, s.burst(r, scope, second, flag, 25*time.Second, time.Second))
		s.rest(cycle.Add(150 * time.Second))
	}
}

func (s *stack) keepTime(r *rand.Rand, scope, account, flag string, fastest, slowest time.Duration, clicks int) bool {
	var dropped bool
	for range clicks {
		s.clock.Advance(fastest + time.Duration(r.Int64N(int64(slowest-fastest))))
		if s.clickAs(scope, account, 100000+uint32(r.IntN(60000)), flag) {
			dropped = true
		}
	}
	return dropped
}

func TestThePairOfOctober13TakingTurnsAtACadenceIsNotBanned(t *testing.T) {
	s := newStack(t, withTheBoundsOfOctober9, productionRetakes)
	s.signIn("desktop", "laptop")

	//nolint:gosec // seeded test PRNG
	r := rand.New(rand.NewPCG(10, 13))

	s.takeTurns(r, polishLine, "PL", 2*time.Hour+30*time.Minute, "desktop", "laptop")
	require.Equal(t, []string{"rotator suspect"}, s.rises)

	assert.False(t, s.keepTime(r, polishLine, "laptop", "PL", 200*time.Millisecond, 580*time.Millisecond, 200))
	assert.Empty(t, s.reports, "in production a cadence beside the turns banned the pair")
	assert.Equal(t, []string{"rotator suspect", "metronome suspect"}, s.rises)

	examination := s.examine(polishLine, "laptop")
	assert.Regexp(t, `^cadence clicks=\d+ median=\S+ spread=3\d\d\.?\d*ms sustained=\S+$`, examination.Readings[2].Evidence)
	assert.Regexp(t, `^rotation accounts=2 `, examination.Readings[9].Evidence)
	assert.Equal(t, 1, examination.Suspects, "turns are never a suspect")
	assert.False(t, examination.Guilty)
}
