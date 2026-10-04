package lead

import (
	"time"
)

type Config struct {
	Margin uint32
	Hold   time.Duration
}

const (
	defaultMargin = 50
	defaultHold   = 30 * time.Second
)

func (c Config) WithDefaults() Config {
	if c.Margin == 0 {
		c.Margin = defaultMargin
	}
	if c.Hold <= 0 {
		c.Hold = defaultHold
	}
	return c
}

type Shares struct {
	tiles map[string]uint32
}

func SharesOf(tiles map[string]uint32) Shares {
	return Shares{tiles: tiles}
}

func (s Shares) Tiles(country string) uint32 { return s.tiles[country] }

// A tie goes to the first code, so two equal countries do not trade the lead read after read.
func (s Shares) Leader() (string, bool) {
	leader := ""
	for country, tiles := range s.tiles {
		if tiles == 0 {
			continue
		}
		best := s.tiles[leader]
		if leader == "" || tiles > best || (tiles == best && country < leader) {
			leader = country
		}
	}
	return leader, leader != ""
}

type Pass struct {
	leader string
	passed string
}

func (p Pass) Leader() string { return p.leader }

func (p Pass) Passed() string { return p.passed }

// A challenger takes the lead once it has led by Margin tiles for Hold, so a flapping lead says nothing.
type Race struct {
	margin uint32
	hold   time.Duration

	leader     string
	challenger string
	since      time.Time
}

func NewRace(config Config) Race {
	config = config.WithDefaults()
	return Race{margin: config.Margin, hold: config.Hold}
}

func (r Race) Leader() string { return r.leader }

func (r Race) Next(shares Shares, at time.Time) (Race, Pass, bool) {
	top, ok := shares.Leader()
	if !ok {
		return r, Pass{}, false
	}

	if r.leader == "" || top == r.leader {
		r.leader, r.challenger = top, ""
		return r, Pass{}, false
	}

	if shares.Tiles(top)-shares.Tiles(r.leader) < r.margin {
		r.challenger = ""
		return r, Pass{}, false
	}

	if r.challenger != top {
		r.challenger, r.since = top, at
	}

	if at.Sub(r.since) < r.hold {
		return r, Pass{}, false
	}

	pass := Pass{leader: top, passed: r.leader}
	r.leader, r.challenger = top, ""
	return r, pass, true
}
