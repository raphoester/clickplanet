package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"
)

// attempt is one click sent. An acked one must be on the map; one with no answer may or may not be.
type attempt struct {
	at      time.Time
	tile    uint32
	country string
	acked   bool
}

type sample struct {
	at time.Time
	ok bool
}

type traffic struct {
	api *api

	players     int
	tilesPerRun uint32
	clickEvery  time.Duration

	mu       sync.Mutex
	attempts []attempt
	clicks   []sample
	probes   []sample
	refused  map[int]int
}

func newTraffic(a *api, players int, tilesPerPlayer uint32, clickEvery time.Duration) *traffic {
	return &traffic{
		api:         a,
		players:     players,
		tilesPerRun: tilesPerPlayer,
		clickEvery:  clickEvery,
		refused:     map[int]int{},
	}
}

// run clicks and probes until ctx ends. Each player owns a range of tiles no other player touches,
// so the last OK click on a tile is the owner the map must end with.
func (t *traffic) run(ctx context.Context) {
	var wg sync.WaitGroup

	for p := range t.players {
		wg.Go(func() { t.play(ctx, p) })
	}
	wg.Go(func() { t.probe(ctx) })

	wg.Wait()
}

func (t *traffic) play(ctx context.Context, player int) {
	rng := rand.New(rand.NewPCG(uint64(player), 7)) //nolint:gosec // a load generator, not a secret.
	ip := fmt.Sprintf("198.51.100.%d", player+1)
	first := uint32(player) * t.tilesPerRun //nolint:gosec // a few dozen players.

	var token string
	for ctx.Err() == nil {
		if token == "" {
			minted, status, err := t.api.createSession(ctx, ip)
			if err != nil {
				t.refuse(status)
				sleep(ctx, 200*time.Millisecond)
				continue
			}
			token = minted
		}

		tile := first + rng.Uint32N(t.tilesPerRun)
		country := countries[rng.IntN(len(countries))]
		status, err := t.api.click(ctx, caller{ip: ip, token: token}, tile, country)
		if ctx.Err() != nil {
			return
		}

		now := time.Now()
		t.mu.Lock()
		t.clicks = append(t.clicks, sample{at: now, ok: err == nil})
		// A refusal with a status never reached the map; no answer at all might have.
		if err == nil || status == 0 {
			t.attempts = append(t.attempts, attempt{at: now, tile: tile, country: country, acked: err == nil})
		}
		t.mu.Unlock()

		switch {
		case err == nil:
		case status == http.StatusUnauthorized:
			token = ""
		default:
			t.refuse(status)
		}

		jitter := time.Duration(rng.Int64N(int64(t.clickEvery / 2)))
		sleep(ctx, t.clickEvery-t.clickEvery/4+jitter)
	}
}

// probe asks for the map density every 100ms: the most basic "is it serving" there is.
func (t *traffic) probe(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, err := t.api.mapDensity(ctx)
			if ctx.Err() != nil {
				return
			}
			t.mu.Lock()
			t.probes = append(t.probes, sample{at: time.Now(), ok: err == nil})
			t.mu.Unlock()
		}
	}
}

func (t *traffic) refuse(status int) {
	t.mu.Lock()
	t.refused[status]++
	t.mu.Unlock()
}

func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

// tileHistory is what the clicks on one tile allow it to hold at the end.
type tileHistory struct {
	lastAck    *attempt
	unanswered []attempt // sent after lastAck, with no answer
}

// allows says whether owner is a value the tile may end with: its last OK click, or any
// unanswered click after it. With neither, fallback (the seed) is the only one.
func (h tileHistory) allows(owner, fallback string) (ok bool, byUnanswered bool) {
	base := fallback
	if h.lastAck != nil {
		base = h.lastAck.country
	}
	if owner == base {
		return true, false
	}
	for _, a := range h.unanswered {
		if a.country == owner {
			return true, true
		}
	}
	return false, false
}

// histories groups the attempts per tile. Each player clicks its tiles one at a time, so time orders them.
func (t *traffic) histories() map[uint32]tileHistory {
	t.mu.Lock()
	defer t.mu.Unlock()

	byTile := make(map[uint32]tileHistory)
	for i := range t.attempts {
		a := t.attempts[i]
		h := byTile[a.tile]
		if a.acked {
			h.lastAck = &t.attempts[i]
			h.unanswered = nil
		} else {
			h.unanswered = append(h.unanswered, a)
		}
		byTile[a.tile] = h
	}

	return byTile
}

func (t *traffic) acked() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	count := 0
	for _, a := range t.attempts {
		if a.acked {
			count++
		}
	}
	return count
}

// outage is the longest stretch with no successful sample, and how many samples failed in all.
type outage struct {
	longest       time.Duration
	from, to      time.Time
	failed, total int
}

func outageOf(samples []sample) outage {
	var o outage
	var lastOK time.Time

	for _, s := range samples {
		o.total++
		if !s.ok {
			o.failed++
			continue
		}
		if !lastOK.IsZero() {
			if gap := s.at.Sub(lastOK); gap > o.longest {
				o.longest, o.from, o.to = gap, lastOK, s.at
			}
		}
		lastOK = s.at
	}

	return o
}
