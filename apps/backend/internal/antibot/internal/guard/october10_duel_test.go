package guard_test

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	household = "2001:db8:10:10::/64"
	rival     = "203.0.113.24"
)

type exchange struct {
	tile  uint32
	at    time.Time
	takes int
}

func (s *stack) reachable(random *rand.Rand, exchanges []*exchange, takes int, from, to time.Duration) *exchange {
	var reachable []*exchange
	for _, e := range exchanges {
		if age := s.clock.Now().Sub(e.at); e.takes == takes && age >= from && age <= to {
			reachable = append(reachable, e)
		}
	}
	if len(reachable) == 0 {
		return nil
	}
	return reachable[random.IntN(len(reachable))]
}

func (s *stack) take(e *exchange, scope, account, flag string) {
	require.False(s.t, s.clickAs(scope, account, e.tile, flag), "a hand must never be dropped")
	e.at, e.takes = s.clock.Now(), e.takes+1
}

func (s *stack) chase(random *rand.Rand, line []uint32) {
	var (
		exchanges  []*exchange
		rivalAt    = s.clock.Now()
		duellistAt = rivalAt.Add(300 * time.Millisecond)
	)

	for swept := 0; swept < len(line); {
		if duellistAt.Before(rivalAt) {
			s.clock.Advance(duellistAt.Sub(s.clock.Now()))
			duellistAt = duellistAt.Add(time.Duration(340+random.IntN(120)) * time.Millisecond)

			if lost := s.reachable(random, exchanges, 1, 190*time.Millisecond, 980*time.Millisecond); lost != nil {
				s.take(lost, household, "duellist", "FR")
			}
			continue
		}

		s.clock.Advance(rivalAt.Sub(s.clock.Now()))
		rivalAt = rivalAt.Add(time.Duration(195+random.IntN(90)) * time.Millisecond)

		var next *exchange
		if random.IntN(2) == 0 {
			next = s.reachable(random, exchanges, 2, 100*time.Millisecond, time.Second)
		}
		if next == nil {
			next = &exchange{tile: line[swept]}
			exchanges = append(exchanges, next)
			swept++
		}
		s.take(next, rival, "rival", "DE")
	}
}

func TestAChaseAlongALineBansNobody(t *testing.T) {
	s := newStack(t, productionRetakes)

	//nolint:gosec // seeded test PRNG
	random := rand.New(rand.NewPCG(10, 768))

	for i := range uint32(2) {
		s.owner[140000+i] = "FR"

		s.clock.Advance(20 * time.Second)
		s.click("198.51.100.9", 140000+i, "DE")

		s.clock.Advance(time.Duration(300+random.IntN(400)) * time.Millisecond)
		require.False(t, s.clickAs(household, "sibling", 140000+i, "FR"))
	}

	s.clock.Advance(4 * time.Minute)

	line := make([]uint32, 30)
	for i := range line {
		line[i] = 131000 + uint32(i)
		s.owner[line[i]] = "FR"
	}

	s.chase(random, line)

	for i, account := range []string{"sibling", "parent"} {
		s.clock.Advance(30 * time.Second)
		require.False(t, s.clickAs(household, account, 150000+uint32(i), "FR"), "nor the accounts behind its address, on their next click")
	}

	assert.Empty(t, s.reports)
	assert.Equal(t, []string{"retaker suspect", "retaker suspect"}, s.rises, "each hand reads as the other does, and no other watchdog reads either")

	reading := s.examine(household, "duellist").Readings[0]
	assert.Equal(t, "suspect", reading.Level)
	assert.Regexp(t, `^roam .* tiles=30$`, reading.Evidence, "thirty tiles with the sibling's two: what read certain")
}
