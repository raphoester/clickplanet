package guard_test

import (
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
)

const (
	oneHome   = "2001:db8:10:9::/64"
	otherHome = "2001:db8:10:77::/64"
)

func withTheBoundsOfOctober9(config *guard.Config) {
	config.Metronome.Detector.MaxGap = 7 * time.Second
	config.Metronome.Detector.MaxSpread = 400 * time.Millisecond

	config.Defender.Enabled = true
	config.Defender.Detector.RetakeWindow = 2 * time.Minute
	config.Defender.Detector.MinClicks = 40
	config.Defender.Detector.MinShare = 0.6
	config.Defender.Detector.TrackWindow = 10 * time.Minute
}

type fighter struct {
	scope   string
	account string
	flag    string
	next    time.Time
}

func byHand(r *rand.Rand) time.Duration {
	return 300*time.Millisecond + time.Duration(r.Int64N(int64(600*time.Millisecond)))
}

func (s *stack) contested(r *rand.Rand, tiles []uint32, flag string) uint32 {
	held := make([]uint32, 0, len(tiles))
	for _, tile := range tiles {
		if s.owner[tile] != flag {
			held = append(held, tile)
		}
	}
	if len(held) == 0 {
		return tiles[r.IntN(len(tiles))]
	}
	return held[r.IntN(len(held))]
}

func (s *stack) fight(r *rand.Rand, lasting time.Duration, fighters ...*fighter) []string {
	tiles := make([]uint32, 60)
	for i := range tiles {
		tiles[i] = 150000 + uint32(i)
	}

	start := s.clock.Now()
	for _, f := range fighters {
		f.next = start.Add(byHand(r))
	}

	var dropped []string
	for {
		f := slices.MinFunc(fighters, func(a, b *fighter) int { return a.next.Compare(b.next) })
		if f.next.Sub(start) >= lasting {
			return dropped
		}
		s.clock.Advance(f.next.Sub(s.clock.Now()))

		if s.clickAs(f.scope, f.account, s.contested(r, tiles, f.flag), f.flag) {
			dropped = append(dropped, f.account)
		}

		f.next = f.next.Add(byHand(r))
	}
}

func households(accounts bool) []*fighter {
	homes := []struct{ scope, flag string }{{oneHome, "SE"}, {otherHome, "NO"}}
	hands := []string{"first", "second", "third"}

	fighters := make([]*fighter, 0, len(homes)*len(hands))
	for _, home := range homes {
		for _, hand := range hands {
			f := &fighter{scope: home.scope, flag: home.flag}
			if accounts {
				f.account = home.flag + "-" + hand
			}
			fighters = append(fighters, f)
		}
	}
	return fighters
}

func TestTheHandsBehindOneNetworkOfOctober9AreNotBanned(t *testing.T) {
	s := newStack(t, withTheBoundsOfOctober9)

	//nolint:gosec // seeded test PRNG
	r := rand.New(rand.NewPCG(10, 9))

	dropped := s.fight(r, 3*time.Minute, households(true)...)

	require.Empty(t, dropped, "in production the network was banned after 1m3s of this")
	assert.Empty(t, s.reports)
	assert.Contains(t, s.rises, "defender suspect", "a tile war still reads as one, on both sides")
	assert.NotContains(t, s.rises, "metronome suspect", "no hand of the six keeps a cadence")
}

func TestWithNoAccountTheSameHandsAreBannedAsOneCaller(t *testing.T) {
	s := newStack(t, withTheBoundsOfOctober9)

	//nolint:gosec // seeded test PRNG
	r := rand.New(rand.NewPCG(10, 9))

	dropped := s.fight(r, 3*time.Minute, households(false)...)

	require.NotEmpty(t, dropped, "three hands on one stream make short gaps, and a short gap has a small spread")
	require.NotEmpty(t, s.reports)
	assert.Regexp(t, `^suspect cadence clicks=\d+ median=[12]\d\d\.\d+ms spread=3\d\d\.\d+ms sustained=\S+$`, s.reports[0].Opinions[2].String())
	assert.Regexp(t, `^suspect defence `, s.reports[0].Opinions[3].String())
}
