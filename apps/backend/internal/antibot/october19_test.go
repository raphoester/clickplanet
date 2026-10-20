package antibot_test

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	irishPhone  = "irish-phone"
	swedishLine = "swedish-line"
)

func (s *stack) switchBrowsers(r *rand.Rand, lasting time.Duration) {
	front := make([]uint32, 60)
	for i := range front {
		front[i] = 150000 + uint32(i)
		s.owner[front[i]] = "IE"
	}

	var (
		browsers = [2]string{"brave", "chrome"}
		start    = s.clock.Now()
		phone    = start.Add(byHand(r))
		rival    = start.Add(3 * time.Minute)
		turn     int
		bank     = 20 + r.IntN(40)
	)

	for s.clock.Now().Sub(start) < lasting {
		if rival.Before(phone) {
			s.clock.Advance(rival.Sub(s.clock.Now()))
			require.False(s.t, s.clickAs(rivalLine, "rival", s.contested(r, front, "GB"), "GB"))
			rival = rival.Add(byHand(r) + byHand(r))
			continue
		}

		s.clock.Advance(phone.Sub(s.clock.Now()))
		tile := s.contested(r, front, "IE")
		if r.IntN(100) < 30 {
			tile = 160000 + uint32(r.IntN(60000))
		}
		require.False(s.t, s.clickAs(irishPhone, browsers[turn%2], tile, "IE"), "a player must never be dropped")

		phone = phone.Add(byHand(r))
		if bank--; bank == 0 {
			turn++
			bank = 20 + r.IntN(40)
			phone = phone.Add(5*time.Second + time.Duration(r.Int64N(int64(7*time.Second))))
		}
	}
}

func TestThePhoneOfOctober19SwitchingBrowsersInATileWarIsNotBanned(t *testing.T) {
	s := newStack(t, withTheBoundsOfOctober9, productionRetakes)
	s.signIn("brave", "chrome")

	//nolint:gosec // seeded test PRNG
	r := rand.New(rand.NewPCG(10, 19))

	s.switchBrowsers(r, 2*time.Hour+10*time.Minute)

	assert.Empty(t, s.reports, "in production the phone was banned when the turns crossed the war")
	assert.Contains(t, s.rises, "defender suspect")
	assert.Contains(t, s.rises, "rotator suspect")

	examination := s.examine(irishPhone, "chrome")
	assert.Regexp(t, `^defence clicks=\d+ retakes=\d+ share=0\.6\d*$`, examination.Readings[3].Evidence)
	assert.Regexp(t, `^rotation accounts=2 byChance=\d+ flag=IE sharedSeconds=0 sharedSlots=2\d$`, examination.Readings[9].Evidence)
	assert.Equal(t, 1, examination.Suspects, "turns are never a suspect")
	assert.False(t, examination.Guilty)
}

func TestThePCOfOctober19AtACadenceTheMorningAfterItsTurnsIsNotBanned(t *testing.T) {
	s := newStack(t, withTheBoundsOfOctober9, productionRetakes)
	s.signIn("first", "second")

	//nolint:gosec // seeded test PRNG
	r := rand.New(rand.NewPCG(10, 19))

	s.takeTurns(r, swedishLine, "SE", 2*time.Hour+30*time.Minute, "first", "second")
	s.rest(s.clock.Now().Add(16 * time.Hour))

	assert.False(t, s.keepTime(r, swedishLine, "second", "SE", 230*time.Millisecond, 700*time.Millisecond, 130))
	assert.Empty(t, s.reports, "in production one account at the bound banned it, on the turns of the day before")
	assert.Equal(t, []string{"rotator suspect", "rotator suspect", "metronome suspect"}, s.rises)

	examination := s.examine(swedishLine, "second")
	assert.Regexp(t, `^cadence clicks=1\d\d median=4\d\d\.?\d*ms spread=3\d\d\.?\d*ms sustained=\S+$`, examination.Readings[2].Evidence)
	assert.Regexp(t, `^rotation accounts=2 byChance=\d+ flag=SE sharedSeconds=0 sharedSlots=\d+$`, examination.Readings[9].Evidence)
	assert.Equal(t, 1, examination.Suspects)
	assert.False(t, examination.Guilty)
}
