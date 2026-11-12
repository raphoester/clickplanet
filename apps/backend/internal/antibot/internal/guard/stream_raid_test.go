package guard_test

import (
	"container/heap"
	"fmt"
	"maps"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

func atSeededDefault(config *guard.Config, set *antibotv1.Rules) {
	proto.Reset(set)
	proto.Merge(set, rules.Seeded().Rules)

	config.Retaker.TrackWindow = 15 * time.Minute
	config.Defender.TrackWindow = 10 * time.Minute
	config.Catcher.TrackWindow = time.Hour
	config.Cohort.TrackWindow = 30 * time.Minute
}

// Production's default since 2026-11-12.
func cohortAndChurnerOff(_ *guard.Config, set *antibotv1.Rules) {
	set.Cohort.Enabled = proto.Bool(false)
	set.Churner.Enabled = proto.Bool(false)
}

type network string

const (
	allOver    network = "from all over"
	carrierV32 network = "one carrier /32"
	carrierV44 network = "one carrier /44"
	carrierV24 network = "one carrier /24"
	carrierNAT network = "one carrier NAT address"
	walker     network = "a bot walking the tile ids"
)

type viewer struct {
	network network
	scope   string
	account string
	first   time.Time
	leaves  time.Time
	next    time.Time
	area    uint32
	steps   uint32

	bank   float64
	filled time.Time
}

func chat(r *rand.Rand, gone time.Time, counts map[network]int) []*viewer {
	var viewers []*viewer
	hosts := cpcolls.NewSet[int]()

	for _, n := range []network{allOver, carrierV32, carrierV44, carrierV24, carrierNAT, walker} {
		for range counts[n] {
			var scope string
			switch n {
			case allOver, walker:
				scope = fmt.Sprintf("2001:%x:%x:%x::/64", r.IntN(0x10000), r.IntN(0x10000), r.IntN(0x10000))
			case carrierV32:
				scope = fmt.Sprintf("2a0d:e488:%x:%x::/64", r.IntN(0x10000), r.IntN(0x10000))
			case carrierV44:
				scope = fmt.Sprintf("2a0d:e487:123%x:%x::/64", r.IntN(0x10), r.IntN(0x10000))
			case carrierV24:
				host := 1 + r.IntN(254)
				for hosts.Contains(host) {
					host = 1 + r.IntN(254)
				}
				hosts.Add(host)
				scope = fmt.Sprintf("198.51.100.%d", host)
			case carrierNAT:
				scope = "203.0.113.77"
			}

			first := gone.Add(time.Duration(r.Int64N(int64(30 * time.Second))))
			stays := 3*time.Minute + time.Duration(r.Int64N(int64(12*time.Minute)))
			if n == walker {
				stays = 15 * time.Minute
			}
			viewers = append(viewers, &viewer{
				network: n,
				scope:   scope,
				account: fmt.Sprintf("guest-%d", len(viewers)),
				first:   first,
				leaves:  first.Add(stays),
				next:    first,
				area:    100000 + uint32(r.IntN(50000)),
				bank:    newGuestBank,
				filled:  first,
			})
		}
	}

	return viewers
}

// Production's clicks on 2026-11-11: a guest's bank and its network's guests' bank, 60 each, at 0.4/s in a fast start.
const (
	newGuestBank = 60
	fastStart    = 30 * time.Minute
)

func refill(tokens float64, filled, now, made time.Time) float64 {
	pace := 0.2
	if now.Sub(made) < fastStart {
		pace = 0.4
	}
	return min(newGuestBank, tokens+now.Sub(filled).Seconds()*pace)
}

type queue []*viewer

func (q queue) Len() int           { return len(q) }
func (q queue) Less(i, j int) bool { return q[i].next.Before(q[j].next) }
func (q queue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q *queue) Push(x any)        { *q = append(*q, x.(*viewer)) }
func (q *queue) Pop() any {
	old := *q
	v := old[len(old)-1]
	*q = old[:len(old)-1]
	return v
}

type networkBank struct {
	tokens float64
	filled time.Time
}

// A throttled try reaches the watchdogs as planet tells it, and a refused one reaches none.
func (s *stack) watchStream(r *rand.Rand, flag string, viewers []*viewer) *cpcolls.Set[string] {
	q := queue(append([]*viewer(nil), viewers...))
	heap.Init(&q)

	networks := map[string]*networkBank{}
	dropped := cpcolls.NewSet[string]()

	for q.Len() > 0 {
		v := heap.Pop(&q).(*viewer)

		s.clock.Advance(v.next.Sub(s.clock.Now()))
		now := s.clock.Now()

		tile := v.area + uint32(r.IntN(300))
		switch v.network {
		case walker:
			v.next = now.Add(2480*time.Millisecond + time.Duration(r.Int64N(int64(40*time.Millisecond))))
			tile = v.area + v.steps
			v.steps++
		case allOver, carrierV32, carrierV44, carrierV24, carrierNAT:
			v.next = now.Add(250*time.Millisecond + time.Duration(r.Int64N(int64(900*time.Millisecond))))
			if r.IntN(25) == 0 {
				v.next = v.next.Add(3*time.Second + time.Duration(r.Int64N(int64(20*time.Second))))
			}
		}
		if !v.next.After(v.leaves) {
			heap.Push(&q, v)
		}

		if s.banned(v.scope, v.account) {
			dropped.Add(v.account)
			continue
		}

		shared, ok := networks[v.scope]
		if !ok {
			shared = &networkBank{tokens: newGuestBank, filled: now}
			networks[v.scope] = shared
		}
		v.bank, v.filled = refill(v.bank, v.filled, now, v.first), now
		shared.tokens, shared.filled = refill(shared.tokens, shared.filled, now, v.first), now

		if v.bank < 1 || shared.tokens < 1 {
			held := s.owner[tile]
			s.guard.Attempted(guard.Click{
				Scope: v.scope, Account: v.account, Tile: tile, Country: flag, At: now, Held: held, NoOp: held == flag,
			})
			continue
		}
		v.bank--
		shared.tokens--

		if s.clickAs(v.scope, v.account, tile, flag) {
			dropped.Add(v.account)
		}
	}

	return dropped
}

func bannedOn(viewers []*viewer, dropped *cpcolls.Set[string]) map[network]int {
	out := map[network]int{}
	for _, v := range viewers {
		if dropped.Contains(v.account) {
			out[v.network]++
		}
	}
	return out
}

func (s *stack) rulesThatBanned(network network, viewers []*viewer) []string {
	scopes := cpcolls.NewSet[string]()
	for _, v := range viewers {
		if v.network == network {
			scopes.Add(v.scope)
		}
	}

	fired := cpcolls.NewSet[string]()
	for _, report := range s.reports {
		if !scopes.Contains(report.Scope) {
			continue
		}
		for _, opinion := range report.Opinions {
			if opinion.Fired() {
				fired.Add(opinion.Evidence.Rule)
			}
		}
	}

	rules := make([]string, 0, fired.Len())
	fired.ForEach(func(rule string) { rules = append(rules, rule) })
	return rules
}

// "Go!": 160 new guests of one flag inside 30 seconds, some put close together by one mobile carrier.
var streamChat = map[network]int{allOver: 64, carrierV32: 60, carrierV44: 12, carrierV24: 12, carrierNAT: 12}

func TestAStreamsChatOnOneCarrierIsBannedByCohortAndChurner(t *testing.T) {
	s := newStack(t, atSeededDefault)

	//nolint:gosec // seeded test PRNG
	r := rand.New(rand.NewPCG(40, 1))
	viewers := chat(r, s.clock.Now().Add(time.Second), streamChat)

	banned := bannedOn(viewers, s.watchStream(r, "fr", viewers))

	assert.Zero(t, banned[allOver], "a raid from all over is only suspect, as TestARaidFromAllOverIsNeverCertain pins")

	assert.Positive(t, banned[carrierV44], "pairs from one /44 that start in the same five seconds, three times, are a chain")
	assert.Contains(t, s.rulesThatBanned(carrierV44, viewers), "chain")
	assert.Positive(t, banned[carrierV24], "so are addresses of one /24")
	assert.Contains(t, s.rulesThatBanned(carrierV24, viewers), "chain")

	assert.Equal(t, 12, banned[carrierNAT], "ten guests born behind one address read churn, and every viewer reads lockstep: the address falls")
	assert.ElementsMatch(t, []string{"lockstep", "churn"}, s.rulesThatBanned(carrierNAT, viewers))

	assert.Positive(t, banned[carrierV32], "a pause in a viewer's first seconds reads as an account that stopped, and the next viewer as its relay")
	assert.ElementsMatch(t, []string{"lockstep", "relay"}, s.rulesThatBanned(carrierV32, viewers))
}

func TestAStreamsChatPlaysFreeWithCohortAndChurnerOff(t *testing.T) {
	s := newStack(t, atSeededDefault, cohortAndChurnerOff)

	//nolint:gosec // seeded test PRNG
	r := rand.New(rand.NewPCG(40, 1))
	chatAndBot := maps.Clone(streamChat)
	chatAndBot[walker] = 1
	viewers := chat(r, s.clock.Now().Add(time.Second), chatAndBot)

	banned := bannedOn(viewers, s.watchStream(r, "fr", viewers))

	assert.Equal(t, map[network]int{walker: 1}, banned, "no viewer is banned, and the other watchdogs still ban a bot")
	require.Len(t, s.reports, 1)
	assert.ElementsMatch(t, []string{"stride", "cadence"}, s.rulesThatBanned(walker, viewers))
	assert.Less(t, s.reports[0].ActiveFor, 6*time.Minute)
}
