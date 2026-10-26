package guard_test

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
)

const (
	sharedAddress = "198.51.100.23"
	homeLine      = "2001:db8:7:9::/64"
)

func (s *stack) signIn(accounts ...string) {
	s.linked.Add(accounts...)
}

func (s *stack) burst(r *rand.Rand, scope, account, flag string, lasting, gap time.Duration) bool {
	var dropped bool
	for until := s.clock.Now().Add(lasting); s.clock.Now().Before(until); {
		if s.clickAs(scope, account, 100000+uint32(r.IntN(60000)), flag) {
			dropped = true
		}
		s.clock.Advance(gap/2 + time.Duration(r.Int64N(int64(gap))))
	}
	return dropped
}

func (s *stack) rest(until time.Time) {
	s.clock.Advance(until.Sub(s.clock.Now()))
}

func (s *stack) rotate(r *rand.Rand, lasting time.Duration) {
	for start := s.clock.Now(); s.clock.Now().Sub(start) < lasting; {
		cycle := s.clock.Now()

		require.False(s.t, s.burst(r, sharedAddress, "desktop-one", "PL", 25*time.Second, 500*time.Millisecond))
		s.rest(cycle.Add(45 * time.Second))
		require.False(s.t, s.burst(r, sharedAddress, "desktop-two", "PL", 25*time.Second, 500*time.Millisecond))
		s.rest(cycle.Add(90 * time.Second))
		require.False(s.t, s.burst(r, sharedAddress, "phone", "PL", 9*time.Second, time.Second))
		s.rest(cycle.Add(150 * time.Second))
	}
}

func TestTheTrioThatTakesTurnsIsBannedTogether(t *testing.T) {
	s := newStack(t)
	s.signIn("desktop-one", "desktop-two", "phone", "student")

	//nolint:gosec // seeded test PRNG
	r := rand.New(rand.NewPCG(10, 10))

	s.rotate(r, 4*time.Hour)
	require.Empty(t, s.reports, "taking turns alone bans nobody")

	s.rest(s.clock.Now().Add(20 * time.Minute))

	var swept bool
	for tile := uint32(180000); tile < 180400 && !swept; tile++ {
		s.clock.Advance(time.Second)
		swept = s.clickAs(sharedAddress, "desktop-one", tile, "PL")
	}
	require.True(t, swept, "a sweep on one account of the ring is two watchdogs")

	assert.True(t, s.clickAs(sharedAddress, "desktop-two", 1, "PL"))
	assert.True(t, s.clickAs(sharedAddress, "phone", 1, "PL"), "the phone never swept, and is banned with the hand that holds it")

	banned := map[string]string{}
	for _, report := range s.reports {
		banned[report.Account] = report.Alongside
	}
	assert.Len(t, banned, 3)
	assert.Empty(t, banned["desktop-one"])
	assert.Equal(t, "desktop-one", banned["desktop-two"])
	assert.Equal(t, "desktop-one", banned["phone"])

	verdicts := s.verdicts(sharedAddress)
	assert.Equal(t, detect.Suspect, verdicts["rotator"])
	assert.Equal(t, detect.Suspect, verdicts["sequencer"])
	assert.Equal(t, detect.Suspect, verdicts["metronome"])

	assert.Empty(t, s.bans.Stored(), "three signed-in accounts leave their address alone")
	assert.False(t, s.clickAs(sharedAddress, "student", 1, "PL"), "so the school behind it still plays")
	assert.True(t, s.clickAs(homeLine, "phone", 1, "PL"), "and the phone is dropped on any network")
}

func TestTakingTurnsAloneBansNobody(t *testing.T) {
	s := newStack(t)
	s.signIn("desktop-one", "desktop-two", "phone")

	//nolint:gosec // seeded test PRNG
	r := rand.New(rand.NewPCG(10, 11))

	s.rotate(r, 4*time.Hour)

	assert.Empty(t, s.reports)
	assert.Equal(t, []string{"rotator suspect"}, s.rises, "it reads, for the jury to cross")

	examination := s.examine(sharedAddress, "")
	require.Len(t, examination.Readings, 9)
	reading := examination.Readings[8]
	assert.Equal(t, "rotator", reading.Watchdog)
	assert.Equal(t, "suspect", reading.Level)
	assert.Regexp(t, `^rotation accounts=3 byChance=\d+ flag=PL sharedSeconds=0 sharedSlots=\d+$`, reading.Evidence)
}

func TestTwoPlayersSideBySideAreNeverRead(t *testing.T) {
	s := newStack(t)
	s.signIn("parent", "child")

	//nolint:gosec // seeded test PRNG
	r := rand.New(rand.NewPCG(10, 12))

	type hand struct {
		account string
		next    time.Time
		until   time.Time
	}
	hands := []*hand{{account: "parent"}, {account: "child"}}

	start := s.clock.Now()
	for s.clock.Now().Sub(start) < 4*time.Hour {
		now := s.clock.Now()
		for _, h := range hands {
			if now.After(h.until) {
				h.next = now.Add(time.Duration(30+r.IntN(77)) * time.Second)
				h.until = h.next.Add(time.Duration(5+r.IntN(21)) * time.Second)
			}
			if !now.Before(h.next) {
				require.False(t, s.clickAs(homeLine, h.account, 100000+uint32(r.IntN(60000)), "FR"))
			}
		}
		s.clock.Advance(time.Second)
	}

	assert.Empty(t, s.reports)
	assert.NotContains(t, s.rises, "rotator suspect")
}
