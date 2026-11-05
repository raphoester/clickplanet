package guard_test

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	newcomerLine = "newcomer-line"
	rivalLine    = "rival-line"
)

func (s *stack) playHard(r *rand.Rand, lasting time.Duration) {
	tile := uint32(110000)

	for start := s.clock.Now(); s.clock.Now().Sub(start) < lasting; {
		for range 15 + r.IntN(25) {
			s.clock.Advance(time.Duration(300+r.IntN(1200)) * time.Millisecond)

			tile = uint32(int(tile) + r.IntN(80) - 40)
			require.False(s.t, s.clickAs(newcomerLine, "newcomer", tile, "PT"), "a player must never be dropped")
		}

		s.clock.Advance(time.Duration(20+r.IntN(100)) * time.Second)
	}
}

func (s *stack) defend(r *rand.Rand, lasting time.Duration) {
	front := make([]uint32, 30)
	for i := range front {
		front[i] = 150000 + uint32(i)
		s.owner[front[i]] = "PT"
	}

	for start := s.clock.Now(); s.clock.Now().Sub(start) < lasting; {
		lost := s.contested(r, front, "ES")

		s.clock.Advance(time.Duration(2000+r.IntN(3000)) * time.Millisecond)
		require.False(s.t, s.clickAs(rivalLine, "rival", lost, "ES"))

		s.clock.Advance(time.Duration(1500+r.IntN(2500)) * time.Millisecond)
		require.False(s.t, s.clickAs(newcomerLine, "newcomer", lost, "PT"), "a player must never be dropped")

		if r.IntN(100) < 55 {
			s.clock.Advance(time.Duration(1000+r.IntN(2000)) * time.Millisecond)
			require.False(s.t, s.clickAs(newcomerLine, "newcomer", 160000+uint32(r.IntN(60000)), "PT"))
		}
	}
}

func TestTheNewcomerOfOctober12WhoPlaysHardAndDefendsIsNotBanned(t *testing.T) {
	s := newStack(t, withTheBoundsOfOctober9, productionRetakes)
	s.signIn("newcomer", "rival")

	//nolint:gosec // seeded test PRNG
	r := rand.New(rand.NewPCG(10, 12))

	s.playHard(r, 5*time.Hour+50*time.Minute)
	s.defend(r, 10*time.Minute)

	assert.Empty(t, s.reports, "in production two suspects banned a human here")
	assert.Equal(t, []string{"metronome suspect", "defender suspect"}, s.rises)

	examination := s.examine(newcomerLine, "newcomer")
	require.Len(t, examination.Readings, 10)
	assert.Equal(t, "suspect", examination.Readings[2].Level)
	assert.Regexp(t, `^stamina busy=5h[345]0m0s clicks=\d+ window=6h0m0s$`, examination.Readings[2].Evidence)
	assert.Equal(t, "suspect", examination.Readings[3].Level)
	assert.Regexp(t, `^defence clicks=1\d\d retakes=\d+ share=0\.6\d*$`, examination.Readings[3].Evidence)
	assert.Equal(t, 1, examination.Suspects, "hours at pace and a tile held are one reading of zeal")
	assert.False(t, examination.Guilty)
}
