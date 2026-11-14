package e2e_test

// Capital Siege night, end to end: the whole backend on a real postgres and Redis, production's rule sets, a day that
// ends as an 8-minute arena battle starts, and the battle ending on the daily cutoff, as tonight's ends at 21:00.
// About 15 minutes on the real clock, so it runs only with SIEGE_NIGHT=1. SIEGE_GEO names a JSON of the map the
// planet module loads ({"n": neighbours by tile, "b": bridge ends by tile, "c": capital tile by flag}), which this
// package cannot load itself; SIEGE_BATTLE sets the battle's length (8m), SIEGE_REPORT a file for the checks.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1/antibotv1connect"
	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1/bonusesv1connect"
	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1/gamemasterv1connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1/quizzesv1connect"
	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1/scoringv1connect"
)

// Production's sets, as read on 2026-11-14 at 17:20 UTC. The antibot's default enforces nothing here: the simulated
// players are scripts, and a ban is judged and counted all the same.
const (
	siegePlanetDefault  = `{"clicks":{"tempo":1, "perSecond":0.2, "burst":60, "scopeMultiplier":10, "v6ScopeMultiplier":4, "guestScopeMultiplier":1, "linkedMultiplier":2, "newAccountClicks":60, "fastStart":"1800s"}, "toll":{"steps":[{"share":0.1, "slowdown":1.5}, {"share":0.2, "slowdown":2.5}, {"share":0.3, "slowdown":4}]}, "frozen":false, "allegianceLocked":false, "arena":{"on":false, "freeze":"8s"}, "seed":{"capitals":{"shields":100}}, "reach":false, "capitals":{"knockOut":false, "fixedShields":false}, "charges":{"spread":{"clicks":8, "rings":1, "unlimited":false}, "bomb":{"rings":4, "off":false}, "enclose":{"maxTiles":25, "held":3, "unlimited":false, "off":false}, "shield":{"held":60, "perTile":10}}}`
	siegePlanetSet      = `{"clicks":{"tempo":3}, "toll":{"steps":[{"share":0.3, "slowdown":1.5}, {"share":0.4, "slowdown":2.5}, {"share":0.5, "slowdown":4}]}, "allegianceLocked":true, "arena":{"on":true, "freeze":"8s"}, "seed":{"capitals":{"shields":50}}, "reach":true, "capitals":{"knockOut":true, "fixedShields":true}, "charges":{"spread":{"clicks":50, "rings":2, "unlimited":true}, "bomb":{"rings":8, "off":true}, "enclose":{"maxTiles":500, "held":8, "off":true}, "shield":{"held":75}}}`
	siegeOffersDefault  = `{"boxes":{"minInterval":"240s", "maxInterval":"480s", "maxChargesPerHour":12}, "kinds":{"refill":5, "spreadClicks":3, "encloseClicks":2, "bomb":1, "shields":4}, "offerTtl":"15s", "missRetry":"120s", "activeWithin":"120s", "spread":{"maxPerBox":4}, "enclose":{"maxPerBox":3}, "shield":{"minPerBox":10, "maxPerBox":30}, "welcome":{"within":"600s", "boxAfter":"60s"}, "signInRefill":true, "frozen":false}`
	siegeOffersSet      = `{"boxes":{"minInterval":"20s", "maxInterval":"40s", "maxChargesPerHour":9999}, "kinds":{"refill":1, "spreadClicks":0, "encloseClicks":0, "shields":1}, "missRetry":"30s", "spread":{"maxPerBox":25}, "enclose":{"maxPerBox":5}, "shield":{"minPerBox":20, "maxPerBox":44}, "welcome":{"boxAfter":"30s"}}`
	siegeQuizDefault    = `{"enabled":true, "minInterval":"240s", "maxInterval":"480s", "missRetry":"120s", "offerTtl":"25s", "answerWindow":"8s", "leaderBias":1, "maxRightAnswersPerHour":12, "target":0.7, "width":1, "floor":0.42, "floorAfter":15, "explore":0.1, "activeWithin":"120s", "welcome":{"within":"600s"}, "frozen":false}`
	siegeQuizSet        = `{"minInterval":"30s", "maxInterval":"60s", "missRetry":"60s", "maxRightAnswersPerHour":9999}`
	siegeAntibotDefault = `{"bans":{"enforce":false, "banDurations":["86400s", "604800s", "94608000s"], "reflagInterval":"300s"}, "jury":{"minSuspects":2, "suspicionWindow":"600s"}, "retaker":{"enabled":true, "reactionWindow":"5s", "minReactions":20, "maxSpread":"0.300s", "maxMedian":"0.250s", "minTiles":15, "roamMedian":"0.600s", "certainTiles":0}, "sequencer":{"enabled":true, "minSteps":40, "minShare":0.75, "certainSteps":200, "certainShare":0.95}, "metronome":{"enabled":true, "maxGap":"7s", "maxSpread":"0.400s", "minClicks":120, "certainFor":"1800s", "certainClicks":900, "shape":{"maxGap":"10s", "clicks":500, "certainClicks":1000}, "clock":{"period":"1s", "clicks":120, "certainClicks":600, "certainFor":"1800s"}, "stamina":{"clicks":40, "window":"21600s", "minBusy":"19800s", "certainBusy":"0s"}}, "defender":{"enabled":true, "retakeWindow":"120s", "minClicks":40, "minShare":0.6, "certainClicks":200, "certainShare":0}, "catcher":{"enabled":true, "minCatches":5, "maxMedian":"3s", "certainMedian":"1.500s", "foreign":{"window":"3600s", "minClaims":0, "certainClaims":0}}, "cohort":{"enabled":false, "startWindow":"5s", "minClicks":20, "minFlagShare":0.9, "rateRatio":1.5, "lengthRatio":1.2, "quietAfter":"60s", "minMembers":2, "certainCohorts":3, "certainMembers":6, "chainWindow":"1800s"}, "scraper":{"enabled":false, "minMaps":5, "certainMaps":15, "certainOffMap":2}, "churner":{"enabled":false, "window":"3600s", "v6":{"minAccounts":4, "certainAccounts":6}, "v4":{"minAccounts":10, "certainAccounts":20}, "relay":{"handoff":"90s", "maxLife":"300s", "minClicks":20, "minFlagShare":0.9, "minLinks":3, "certainLinks":6}}, "hopper":{"enabled":true, "minAngle":45, "minGap":"0.100s", "maxGap":"30s", "minSteps":40, "minShare":0.4, "certainSteps":200, "certainShare":0}, "rotator":{"enabled":true, "window":"86400s", "minSeconds":15, "minByChance":90, "maxShare":0.05, "minSlots":24, "certainSlots":48, "certainByChance":300}}`
	siegeAntibotSet     = `{"retaker":{"minTiles":0}, "defender":{"enabled":false}, "catcher":{"minCatches":12}, "cohort":{"certainMembers":50}}`
	siegeDescription    = "Capital Siege: the map starts blank and every country starts on its capital. Cross the sea on the bridges. Take a capital to knock its flag out. The map comes back after the battle."
	siegeTiles          = 260503
)

type siegeGeo struct {
	N [][]uint32        `json:"n"`
	B [][]uint32        `json:"b"`
	C map[string]uint32 `json:"c"`

	mu    sync.Mutex
	dists map[uint32][]uint16
}

func (g *siegeGeo) dist(from uint32) []uint16 {
	g.mu.Lock()
	defer g.mu.Unlock()
	if d, ok := g.dists[from]; ok {
		return d
	}
	d := make([]uint16, len(g.N))
	for i := range d {
		d[i] = math.MaxUint16
	}
	d[from] = 0
	q := []uint32{from}
	for len(q) > 0 {
		x := q[0]
		q = q[1:]
		for _, set := range [][]uint32{g.N[x], g.B[x]} {
			for _, y := range set {
				if d[y] == math.MaxUint16 {
					d[y] = d[x] + 1
					q = append(q, y)
				}
			}
		}
	}
	g.dists[from] = d
	return d
}

// siegeMap is a map as the wire says it: a flag and shields per tile.
type siegeMap struct {
	owner   []string
	shields []uint8
	out     []string
	epoch   uint32
	seq     uint64
}

func decodeMap(res *planetv1.GetMapResponse) siegeMap {
	m := siegeMap{owner: make([]string, siegeTiles+1), shields: make([]uint8, siegeTiles+1), out: res.GetKnockedOut(),
		epoch: res.GetVersion().GetEpoch(), seq: res.GetVersion().GetSeq()}
	codes, tiles, start := res.GetCodes(), res.GetTiles(), res.GetStartTileId()
	for i := 0; i+1 < len(tiles); i += 2 {
		tile := int(start) + i/2
		if tile <= siegeTiles {
			m.owner[tile] = codes[int(tiles[i])|int(tiles[i+1])<<8]
		}
	}
	for _, s := range res.GetShields() {
		m.shields[s.GetTileId()] = uint8(s.GetShields())
	}
	return m
}

func (m siegeMap) holdings() map[string]int {
	h := map[string]int{}
	for _, o := range m.owner {
		if o != "" {
			h[o]++
		}
	}
	return h
}

// mirror is the arena's map as the observer's stream tells it, from the snapshot taken in the start's freeze.
type mirror struct {
	mu      sync.RWMutex
	armed   bool
	m       siegeMap
	byFlag  map[string]map[uint32]struct{}
	outBy   map[string]string
	applied int
	stale   int
}

func (r *mirror) arm(m siegeMap) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m, r.byFlag, r.outBy, r.armed = m, map[string]map[uint32]struct{}{}, map[string]string{}, true
	for t, o := range m.owner {
		if o != "" {
			r.own(o)[uint32(t)] = struct{}{}
		}
	}
}

func (r *mirror) own(flag string) map[uint32]struct{} {
	s, ok := r.byFlag[flag]
	if !ok {
		s = map[uint32]struct{}{}
		r.byFlag[flag] = s
	}
	return s
}

func (r *mirror) set(tile uint32, flag string, shields uint32) {
	if old := r.m.owner[tile]; old != "" {
		delete(r.own(old), tile)
	}
	r.m.owner[tile] = flag
	r.m.shields[tile] = uint8(shields)
	if flag != "" {
		r.own(flag)[tile] = struct{}{}
	}
}

func (r *mirror) update(u *planetv1.TileUpdate) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.armed || u.GetVersion().GetEpoch() != r.m.epoch {
		return
	}
	if u.GetVersion().GetSeq() <= r.m.seq {
		r.stale++
		return
	}
	r.applied++
	r.set(u.GetTileId(), u.GetCountryId(), u.GetShields())
}

func (r *mirror) knockOut(k *planetv1.KnockOut) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.armed || k.GetVersion().GetEpoch() != r.m.epoch {
		return
	}
	for _, t := range k.GetClearedTileIds() {
		r.set(t, "", 0)
	}
	r.m.out = append(r.m.out, k.GetCountryId())
	r.outBy[k.GetCountryId()] = k.GetByCountryId()
}

func (r *mirror) isOut(flag string) (bool, string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	by, out := r.outBy[flag]
	return out, by
}

type siegeLog struct {
	mu     sync.Mutex
	t0     time.Time
	lines  []string
	failed []string
	t      *testing.T
}

func (l *siegeLog) at() string { return fmt.Sprintf("[%s]", time.Now().UTC().Format("15:04:05.0")) }

func (l *siegeLog) note(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	line := l.at() + " " + fmt.Sprintf(format, args...)
	l.lines = append(l.lines, line)
	fmt.Println(line)
}

func (l *siegeLog) check(ok bool, format string, args ...any) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	mark := "PASS"
	if !ok {
		mark = "FAIL"
	}
	line := fmt.Sprintf("%s %s %s", l.at(), mark, fmt.Sprintf(format, args...))
	l.lines = append(l.lines, line)
	fmt.Println(line)
	if !ok {
		l.failed = append(l.failed, line)
		l.t.Errorf("%s", line)
	}
	return ok
}

func refusalOf(err error) string {
	if err == nil {
		return "ok"
	}
	var ce *connect.Error
	if !errors.As(err, &ce) {
		return "error: " + err.Error()
	}
	for _, d := range ce.Details() {
		v, e := d.Value()
		if e != nil {
			continue
		}
		switch a := v.(type) {
		case *planetv1.FrozenRefusal:
			return "frozen"
		case *planetv1.ReachRefusal:
			return "reach"
		case *planetv1.BombRefusal:
			return "bomb"
		case *planetv1.BanRefusal:
			return "banned"
		case *planetv1.Allegiance:
			if a.GetLocked() {
				return "allegiance-locked"
			}
			return "allegiance"
		}
	}
	if ce.Code() == connect.CodeResourceExhausted {
		return "throttled"
	}
	return ce.Code().String() + ": " + ce.Message()
}

type siegePlayer struct {
	f       *fighter
	flag    string
	target  string
	role    string // attack, defend, roam
	aim     float64
	rng     *rand.Rand
	counts  map[string]int
	spreads int
	mu      sync.Mutex
}

func (p *siegePlayer) count(what string) {
	p.mu.Lock()
	p.counts[what]++
	p.mu.Unlock()
}

func (p *siegePlayer) click(ctx context.Context, tile uint32, spread bool) string {
	req := connect.NewRequest(&planetv1.ClickRequest{TileId: tile, CountryId: p.flag, Spread: spread})
	p.f.send(req.Header())
	_, err := planetv1connect.NewClickServiceClient(fast, p.f.stack.baseURL).Click(ctx, req)
	return refusalOf(err)
}

func (p *siegePlayer) pledge(ctx context.Context, flag string) string {
	req := connect.NewRequest(&planetv1.PledgeAllegianceRequest{CountryId: flag})
	p.f.send(req.Header())
	_, err := planetv1connect.NewClickServiceClient(fast, p.f.stack.baseURL).PledgeAllegiance(ctx, req)
	return refusalOf(err)
}

// choose is a hand's next tile: next to the flag's land, or across a bridge, toward its target or anywhere on its front.
func (p *siegePlayer) choose(g *siegeGeo, r *mirror) (uint32, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	own := r.byFlag[p.flag]
	if len(own) == 0 {
		return 0, false
	}
	seen := map[uint32]struct{}{}
	var front []uint32
	for t := range own {
		for _, set := range [][]uint32{g.N[t], g.B[t]} {
			for _, u := range set {
				if r.m.owner[u] == p.flag {
					continue
				}
				if _, ok := seen[u]; !ok {
					seen[u] = struct{}{}
					front = append(front, u)
				}
			}
		}
	}
	if len(front) == 0 {
		return 0, false
	}
	home := g.dist(g.C[p.flag])
	if p.role == "defend" && p.rng.Float64() < 0.6 {
		best, bd := uint32(0), uint16(math.MaxUint16)
		for _, u := range front {
			if o := r.m.owner[u]; o != "" && o != p.flag && home[u] <= 4 && home[u] < bd {
				best, bd = u, home[u]
			}
		}
		if bd != math.MaxUint16 {
			return best, true
		}
	}
	if p.rng.Float64() < p.aim {
		target := p.target
		if _, out := r.outBy[target]; target == "" || out || target == p.flag {
			target, p.target = "", ""
			bd := uint16(math.MaxUint16)
			for code, c := range g.C {
				if _, out := r.outBy[code]; out || code == p.flag {
					continue
				}
				if home[c] < bd {
					bd, target = home[c], code
				}
			}
			p.target = target
		}
		if target != "" {
			dt := g.dist(g.C[target])
			best, bd, ties := uint32(0), uint16(math.MaxUint16), 0
			for _, u := range front {
				if d := dt[u]; d < bd {
					best, bd, ties = u, d, 1
				} else if d == bd {
					ties++
					if p.rng.IntN(ties) == 0 {
						best = u
					}
				}
			}
			if bd != math.MaxUint16 {
				return best, true
			}
		}
	}
	return front[p.rng.IntN(len(front))], true
}

func placesOf(values map[string]int) map[string]int {
	type kv struct {
		k string
		v int
	}
	var all []kv
	for k, v := range values {
		all = append(all, kv{k, v})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].v > all[j].v })
	ranks := map[string]int{}
	for i, e := range all {
		if i > 0 && all[i-1].v == e.v {
			ranks[e.k] = ranks[all[i-1].k]
		} else {
			ranks[e.k] = i + 1
		}
	}
	return ranks
}

var rankPoints = []uint32{25, 18, 15, 12, 10, 8, 6, 4, 2, 1}

func pointsAt(rank int, times uint32) uint32 {
	if rank < 1 || rank > len(rankPoints) {
		return 0
	}
	return rankPoints[rank-1] * times
}

func TestCapitalSiegeNight(t *testing.T) {
	if os.Getenv("SIEGE_NIGHT") == "" {
		t.Skip("set SIEGE_NIGHT=1")
	}
	raw, err := os.ReadFile(os.Getenv("SIEGE_GEO"))
	if err != nil {
		t.Fatal(err)
	}
	g := &siegeGeo{dists: map[uint32][]uint16{}}
	if err := json.Unmarshal(raw, g); err != nil {
		t.Fatal(err)
	}

	// Tonight at scale: the battle ends on the daily cutoff, as 21:00 ends tonight's; the day before ends as it starts.
	battle := 8 * time.Minute
	if v := os.Getenv("SIEGE_BATTLE"); v != "" {
		battle, _ = time.ParseDuration(v)
	}
	T2 := time.Now().UTC().Add(battle + 6*time.Minute).Truncate(time.Minute)
	T1 := T2.Add(-battle)
	dailyCutoff = T2.Sub(T2.Truncate(24 * time.Hour))
	log := &siegeLog{t: t, t0: time.Now()}
	log.note("timeline: day K ends and the battle starts at T1=%s, the battle ends on the cutoff T2=%s (cutoff %s)",
		T1.Format("15:04:05"), T2.Format("15:04:05"), dailyCutoff)

	game := startGame(t)
	ctx := t.Context()
	log.note("backend up: postgres and Redis in Docker, all modules in one process")

	planetAdmin := planetv1connect.NewAdminServiceClient(http.DefaultClient, game.consoleURL)
	offersAdmin := bonusesv1connect.NewAdminServiceClient(http.DefaultClient, game.consoleURL)
	quizAdmin := quizzesv1connect.NewAdminServiceClient(http.DefaultClient, game.consoleURL)
	antibotAdmin := antibotv1connect.NewAdminServiceClient(http.DefaultClient, game.consoleURL)
	schedule := gamemasterv1connect.NewAdminServiceClient(http.DefaultClient, game.consoleURL)
	public := planetv1connect.NewClickServiceClient(fast, game.baseURL)
	turns := gamemasterv1connect.NewGamemasterServiceClient(http.DefaultClient, game.baseURL)
	scoring := scoringv1connect.NewScoringServiceClient(http.DefaultClient, game.baseURL)

	// ---- the backoffice: production's sets in all four modules
	for _, s := range []struct {
		name, js string
	}{{"default", siegePlanetDefault}, {"capital-siege", siegePlanetSet}} {
		rules := &planetv1.Rules{}
		must(t, protojson.Unmarshal([]byte(s.js), rules))
		_, err := planetAdmin.SaveRuleSet(ctx, connect.NewRequest(&planetv1.SaveRuleSetRequest{Set: &planetv1.RuleSet{Name: s.name, Rules: rules}}))
		log.check(err == nil, "planet keeps production's %q set: %v", s.name, err)
	}
	for _, s := range []struct{ name, js string }{{"default", siegeOffersDefault}, {"capital-siege", siegeOffersSet}} {
		rules := &bonusesv1.Rules{}
		must(t, protojson.Unmarshal([]byte(s.js), rules))
		_, err := offersAdmin.SaveRuleSet(ctx, connect.NewRequest(&bonusesv1.SaveRuleSetRequest{Set: &bonusesv1.RuleSet{Name: s.name, Rules: rules}}))
		log.check(err == nil, "the offers keep production's %q set: %v", s.name, err)
	}
	for _, s := range []struct{ name, js string }{{"default", siegeQuizDefault}, {"capital-siege", siegeQuizSet}} {
		rules := &quizzesv1.Rules{}
		must(t, protojson.Unmarshal([]byte(s.js), rules))
		_, err := quizAdmin.SaveRuleSet(ctx, connect.NewRequest(&quizzesv1.SaveRuleSetRequest{Set: &quizzesv1.RuleSet{Name: s.name, Rules: rules}}))
		log.check(err == nil, "the quizzes keep production's %q set: %v", s.name, err)
	}
	for _, s := range []struct{ name, js string }{{"default", siegeAntibotDefault}, {"capital-siege", siegeAntibotSet}} {
		rules := &antibotv1.Rules{}
		must(t, protojson.Unmarshal([]byte(s.js), rules))
		_, err := antibotAdmin.SaveRuleSet(ctx, connect.NewRequest(&antibotv1.SaveRuleSetRequest{Set: &antibotv1.RuleSet{Name: s.name, Rules: rules}}))
		log.check(err == nil, "the antibot keeps production's %q set (bans judged, not enforced): %v", s.name, err)
	}
	added, err := schedule.AddEntry(ctx, connect.NewRequest(&gamemasterv1.AddEntryRequest{
		RuleSet: "capital-siege", Kind: gamemasterv1.EntryKind_ENTRY_KIND_BATTLE, Description: siegeDescription,
		StartsAt: timestamppb.New(T1), EndsAt: timestamppb.New(T2), Priority: 1, Times: 2,
	}))
	if !log.check(err == nil, "the schedule takes the battle, every module keeping the set: %v", err) {
		t.FailNow()
	}
	battleID := added.Msg.GetId()

	turn, err := turns.GetTurn(ctx, connect.NewRequest(&gamemasterv1.GetTurnRequest{}))
	must(t, err)
	dayK := turn.Msg.GetTimeline().GetToday().GetRoundId()
	book := turn.Msg.GetTimeline().GetCurrent().GetBookId()
	log.note("day K is %s (ends %s) in book %s; the battle's round is %s", dayK,
		turn.Msg.GetTimeline().GetToday().GetEndsAt().AsTime().Format("15:04:05"), book, battleID)
	log.check(turn.Msg.GetTimeline().GetToday().GetEndsAt().AsTime().Equal(T1), "day K ends as the battle starts (%s)",
		turn.Msg.GetTimeline().GetToday().GetEndsAt().AsTime().Format("15:04:05"))

	// ---- the players: production's flags, one address each
	type seat struct{ flag, target, role string }
	seats := []seat{
		{"va", "it", "attack"}, {"va", "it", "attack"}, {"at", "hu", "attack"}, {"at", "hu", "attack"},
		{"hu", "", "defend"}, {"de", "pl", "attack"}, {"pl", "de", "attack"}, {"fr", "", "roam"}, {"bg", "", "roam"},
		{"gb", "", "roam"}, {"se", "", "roam"}, {"py", "bo", "attack"}, {"bo", "", "defend"}, {"in", "", "roam"},
		{"sc", "", "roam"},
	}
	var players []*siegePlayer
	for i, s := range seats {
		f := game.fighter(t, fmt.Sprintf("198.51.100.%d", 10+i), s.flag)
		aim := 0.8
		if s.role == "roam" {
			aim = 0.3
		}
		players = append(players, &siegePlayer{f: f, flag: s.flag, target: s.target, role: s.role, aim: aim,
			rng: rand.New(rand.NewPCG(uint64(i), 99)), counts: map[string]int{}})
	}
	probe := game.fighter(t, "198.51.100.200", "fr")
	log.note("%d players and a probe pledged (one Scot, whose flag has no capital)", len(players))

	// ---- the observer: one page's planet stream, kept open as a page keeps it
	type cut struct {
		at  time.Time
		err string
	}
	var (
		obsMu       sync.Mutex
		cuts        []cut
		rulesSeen   []time.Time
		knockOuts   []*planetv1.KnockOut
		spreadSizes []int
		updates     int
	)
	arena := &mirror{}
	cutCh := make(chan time.Time, 8)
	obsCtx, stopObs := context.WithCancel(ctx)
	defer stopObs()
	go func() {
		for obsCtx.Err() == nil {
			stream, err := public.ListenForEvents(obsCtx, connect.NewRequest(&planetv1.ListenForEventsRequest{}))
			if err != nil {
				time.Sleep(200 * time.Millisecond)
				continue
			}
			for stream.Receive() {
				ev := stream.Msg()
				switch e := ev.GetEvent().(type) {
				case *planetv1.PlanetEvent_TileUpdate:
					obsMu.Lock()
					updates++
					obsMu.Unlock()
					arena.update(e.TileUpdate)
				case *planetv1.PlanetEvent_KnockOut:
					obsMu.Lock()
					knockOuts = append(knockOuts, e.KnockOut)
					obsMu.Unlock()
					arena.knockOut(e.KnockOut)
					log.note("stream: knock_out %s by %s (capital %d, %d tiles cleared)", e.KnockOut.GetCountryId(),
						e.KnockOut.GetByCountryId(), e.KnockOut.GetTileId(), len(e.KnockOut.GetClearedTileIds()))
				case *planetv1.PlanetEvent_TilesSpread:
					obsMu.Lock()
					spreadSizes = append(spreadSizes, len(e.TilesSpread.GetSpreadTileIds()))
					obsMu.Unlock()
				case *planetv1.PlanetEvent_RulesChanged:
					obsMu.Lock()
					rulesSeen = append(rulesSeen, time.Now())
					obsMu.Unlock()
				}
			}
			why := fmt.Sprint(stream.Err())
			_ = stream.Close()
			if obsCtx.Err() != nil {
				return
			}
			now := time.Now()
			obsMu.Lock()
			cuts = append(cuts, cut{now, why})
			obsMu.Unlock()
			select {
			case cutCh <- now:
			default:
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	readMap := func() siegeMap {
		res, err := public.GetMap(ctx, connect.NewRequest(&planetv1.GetMapRequest{}))
		must(t, err)
		return decodeMap(res.Msg)
	}

	// ---- day K on the world's map: every player paints around its capital, the default rules, no reach
	stopPaint := T1.Add(-20 * time.Second)
	var wg sync.WaitGroup
	for _, p := range players {
		wg.Add(1)
		go func() {
			defer wg.Done()
			capital, ok := g.C[p.flag]
			if !ok {
				capital = g.C["gb"] // the Scot paints for Scotland around Edinburgh's side of the island
			}
			home := g.dist(capital)
			var near []uint32
			for t, d := range home {
				if d >= 1 && d <= 6 {
					near = append(near, uint32(t))
				}
			}
			n := 15 + p.rng.IntN(30)
			for i := 0; i < n && time.Now().Before(stopPaint); i++ {
				r := p.click(ctx, near[p.rng.IntN(len(near))], false)
				p.count("day:" + r)
				time.Sleep(time.Duration(200+p.rng.IntN(600)) * time.Millisecond)
			}
		}()
	}
	wg.Wait()
	var dayOK int
	for _, p := range players {
		dayOK += p.counts["day:ok"]
	}
	log.check(dayOK > 200, "day K: the players took %d tiles on the world's map", dayOK)

	// The world's map as it stands before the battle: read past GetMap's 5 s cache, with nobody clicking.
	time.Sleep(time.Until(T1.Add(-9 * time.Second)))
	world := readMap()
	worldHeld := world.holdings()
	log.note("world's map before the battle: epoch %d seq %d, %d flags hold %d tiles", world.epoch, world.seq,
		len(worldHeld), sum(worldHeld))
	holdings, err := public.GetHoldings(ctx, connect.NewRequest(&planetv1.GetHoldingsRequest{}))
	must(t, err)
	agree := true
	for _, h := range holdings.Msg.GetCountries() {
		if int(h.GetTiles()) != worldHeld[h.GetCountryId()] {
			agree = false
		}
	}
	log.check(agree && len(holdings.Msg.GetCountries()) == len(worldHeld), "GetHoldings agrees with GetMap on the world's map")

	// ---- T1
	time.Sleep(time.Until(T1))
	log.note("T1: the battle starts")
	var t1Cut time.Time
	select {
	case t1Cut = <-cutCh:
	case <-time.After(30 * time.Second):
	}
	log.check(!t1Cut.IsZero(), "the arena opened and cut every planet stream %.1fs after T1", t1Cut.Sub(T1).Seconds())

	// The freeze, as a page sees it: clicks refused until the battle's set plays.
	var firstOK time.Time
	frozen := 0
	probeTile := g.N[g.C["fr"]][0]
	for time.Since(t1Cut) < 25*time.Second {
		r := (&siegePlayer{f: probe, flag: "fr"}).click(ctx, probeTile, true)
		if r == "frozen" {
			frozen++
		} else if r == "ok" {
			firstOK = time.Now()
			break
		} else {
			log.note("probe in the start's freeze: %s", r)
		}
		time.Sleep(250 * time.Millisecond)
	}
	freezeFor := firstOK.Sub(t1Cut).Seconds()
	log.check(frozen > 0 && !firstOK.IsZero() && freezeFor >= 7 && freezeFor <= 11,
		"the start's freeze refused %d clicks (FrozenRefusal) and lifted %.1fs after the blank", frozen, freezeFor)

	// The arena's map: every capital held by its flag with 50 shields, and nothing else.
	var fresh siegeMap
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		fresh = readMap()
		if fresh.epoch != world.epoch {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	log.check(fresh.epoch != world.epoch, "GetMap answers the arena's map, a new epoch (%d → %d)", world.epoch, fresh.epoch)
	held, wrong := 0, 0
	for tile, o := range fresh.owner {
		if o == "" {
			continue
		}
		held++
		if c, ok := g.C[o]; !ok || (c != uint32(tile) && !(o == "fr" && uint32(tile) == probeTile)) {
			wrong++
		}
	}
	capitalsRight := 0
	for code, c := range g.C {
		if fresh.owner[c] == code && fresh.shields[c] == 50 {
			capitalsRight++
		}
	}
	log.check(capitalsRight == len(g.C) && wrong == 0,
		"the fresh map holds %d tiles: %d/%d capitals held by their flag with 50 shields, %d other tiles (the probe's spread aside)",
		held, capitalsRight, len(g.C), wrong)
	log.check(len(fresh.out) == 0, "no flag is out on the fresh map")
	// Arm the mirror on the capitals alone, from the snapshot of the freeze, and replay nothing older.
	seed := fresh
	arena.arm(seed)

	// The rules: the battle's set in every module, and what a page reads.
	rulesOf := func() (string, string, string, string) {
		pr, _ := planetAdmin.GetRules(ctx, connect.NewRequest(&planetv1.GetRulesRequest{}))
		or, _ := offersAdmin.GetRules(ctx, connect.NewRequest(&bonusesv1.GetRulesRequest{}))
		qr, _ := quizAdmin.GetRules(ctx, connect.NewRequest(&quizzesv1.GetRulesRequest{}))
		ar, _ := antibotAdmin.GetRules(ctx, connect.NewRequest(&antibotv1.GetRulesRequest{}))
		return pr.Msg.GetPlaying(), or.Msg.GetPlaying(), qr.Msg.GetPlaying(), ar.Msg.GetPlaying()
	}
	pp, op, qp, ap := rulesOf()
	log.check(pp == "capital-siege" && op == "capital-siege" && qp == "capital-siege" && ap == "capital-siege",
		"every module plays capital-siege: planet %s, offers %s, quizzes %s, antibot %s", pp, op, qp, ap)
	cr, err := public.GetChargeRules(ctx, connect.NewRequest(&planetv1.GetChargeRulesRequest{}))
	must(t, err)
	c := cr.Msg
	log.check(c.GetReach() && c.GetKnockOut() && c.GetCapitalShieldsFixed() && c.GetSpreadUnlimited() && c.GetBombsOff(),
		"GetChargeRules says reach, knock-out, fixed capital shields, unlimited spread and no bombs: %s", protojson.Format(c))
	obsMu.Lock()
	sawRules := len(rulesSeen) > 0 && rulesSeen[len(rulesSeen)-1].After(t1Cut)
	obsMu.Unlock()
	log.check(sawRules, "the observer's stream, opened again after the cut, heard rules_changed")
	round, err := scoring.GetRound(ctx, connect.NewRequest(&scoringv1.GetRoundRequest{RoundId: battleID}))
	log.check(err == nil && round.Msg.GetRound().GetTimes() == 2 &&
		math.Abs(float64(round.Msg.GetRound().GetStartedAtUnixMs()-T1.UnixMilli())) < 1000,
		"scoring opened the battle's round, ×2, started at T1: %v", err)
	bt, err := turns.GetTurn(ctx, connect.NewRequest(&gamemasterv1.GetTurnRequest{}))
	must(t, err)
	log.check(bt.Msg.GetTurn().GetKind() == gamemasterv1.EntryKind_ENTRY_KIND_BATTLE && bt.Msg.GetTurn().GetRoundId() == battleID,
		"GetTurn tells a battle, its round the entry")

	// Day K, scored at T1 on the world's map: held and gained rank alike from a blank start.
	var dayRound *scoringv1.Round
	for range 50 {
		r, err := scoring.GetRound(ctx, connect.NewRequest(&scoringv1.GetRoundRequest{RoundId: dayK}))
		if err == nil && r.Msg.GetRound().GetEndedAtUnixMs() != 0 {
			dayRound = r.Msg.GetRound()
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if log.check(dayRound != nil, "scoring closed day K") {
		log.check(math.Abs(float64(dayRound.GetEndedAtUnixMs()-T1.UnixMilli())) < 1000, "day K ended at T1")
		want := placesOf(worldHeld)
		bad := 0
		for _, s := range dayRound.GetStandings() {
			if int(s.GetRank()) != want[s.GetCountryId()] || s.GetPoints() != pointsAt(want[s.GetCountryId()], 1) {
				bad++
				log.note("  day K %s: rank %d points %d, expected rank %d points %d", s.GetCountryId(), s.GetRank(),
					s.GetPoints(), want[s.GetCountryId()], pointsAt(want[s.GetCountryId()], 1))
			}
		}
		log.check(bad == 0 && len(dayRound.GetStandings()) == len(want),
			"day K placed %d flags by the tiles they held at T1, points 25/18/15/… (%d off)", len(dayRound.GetStandings()), bad)
	}

	// ---- the battle: each hand plays until the end's freeze, as the page lets it
	stopPlay := T2.Add(-1 * time.Second)
	var scot *siegePlayer
	for _, p := range players {
		if p.flag == "sc" {
			scot = p
		}
	}
	r := scot.click(ctx, g.N[g.C["gb"]][0], true)
	log.check(r == "reach", "the Scot's flag has no capital, so no land: a click is refused (%s)", r)
	r = scot.pledge(ctx, "gb")
	log.check(r == "ok", "the Scot changes flag past the lock, its flag standing nowhere (%s)", r)
	scot.flag = "gb"

	for _, p := range players {
		wg.Add(1)
		go func() {
			defer wg.Done()
			handled := false
			for time.Now().Before(stopPlay) {
				if out, by := arena.isOut(p.flag); out {
					if handled {
						time.Sleep(200 * time.Millisecond)
						continue
					}
					// a flag out: another out flag is refused, the flag that took the capital is allowed
					arena.mu.RLock()
					other := ""
					for f := range arena.outBy {
						if f != p.flag {
							other = f
						}
					}
					arena.mu.RUnlock()
					if other != "" {
						r := p.pledge(ctx, other)
						log.check(r != "ok", "%s's player, knocked out, may not pick %s, out too (%s)", p.flag, other, r)
					}
					was := p.flag
					r := p.pledge(ctx, by)
					log.check(r == "ok", "%s's player, knocked out, picks %s past the lock (%s)", was, by, r)
					if r == "ok" {
						p.flag, p.target = by, ""
					}
					handled = r != "ok"
					continue
				}
				tile, ok := p.choose(g, arena)
				if !ok {
					time.Sleep(200 * time.Millisecond)
					continue
				}
				r := p.click(ctx, tile, true)
				p.count(r)
				switch r {
				case "throttled":
					time.Sleep(time.Duration(300+p.rng.IntN(400)) * time.Millisecond)
				case "frozen":
					time.Sleep(300 * time.Millisecond)
				}
				time.Sleep(time.Duration(80+p.rng.IntN(250)) * time.Millisecond)
			}
		}()
	}

	// What a page tries while the battle plays.
	time.Sleep(20 * time.Second)
	at := players[2]
	r = at.pledge(ctx, "de")
	log.check(r == "allegiance-locked", "an Austrian may not change flag mid-battle (%s)", r)
	far := g.C["au"]
	r = (&siegePlayer{f: players[7].f, flag: "fr"}).click(ctx, far, true)
	log.check(r == "reach" || strings.HasPrefix(r, "throttled"), "a French click on Canberra is out of reach (%s)", r)
	fr := players[7]
	_, err = planetAdmin.GrantCharges(ctx, connect.NewRequest(&planetv1.GrantChargesRequest{AccountId: fr.f.account, Shields: 20, Bomb: true}))
	log.note("the operator grants the French player 20 shields and a bomb: %v", err)
	shield := func(tile uint32) string {
		req := connect.NewRequest(&planetv1.PlaceShieldRequest{TileId: tile, CountryId: "fr"})
		fr.f.send(req.Header())
		_, err := public.PlaceShield(ctx, req)
		return refusalOf(err)
	}
	r = shield(g.C["fr"])
	log.check(r != "ok", "a shield on Paris, a capital, is refused (%s)", r)
	arena.mu.RLock()
	var ownTile uint32
	for tile := range arena.byFlag["fr"] {
		if tile != g.C["fr"] {
			ownTile = tile
			break
		}
	}
	arena.mu.RUnlock()
	if ownTile != 0 {
		r = shield(ownTile)
		log.check(r == "ok", "a shield on another French tile is placed (%s)", r)
	}
	bomb := connect.NewRequest(&planetv1.DropBombRequest{Target: &planetv1.GlobePoint{X: 1}, CountryId: "fr"})
	fr.f.send(bomb.Header())
	_, err = public.DropBomb(ctx, bomb)
	log.check(refusalOf(err) == "bomb", "a bomb held is not dropped (%s)", refusalOf(err))

	// A minute at a time: what the stream and the map say.
	for time.Now().Before(T2.Add(-70 * time.Second)) {
		time.Sleep(60 * time.Second)
		arena.mu.RLock()
		h := map[string]int{}
		for f, s := range arena.byFlag {
			h[f] = len(s)
		}
		outs := len(arena.outBy)
		arena.mu.RUnlock()
		top := topOf(h, 6)
		obsMu.Lock()
		log.note("battle +%.0fs: %d flags out, %d tile updates heard, leaders %s", time.Since(T1).Seconds(), outs, updates, top)
		obsMu.Unlock()
	}

	// ---- the end's freeze: from 8 s before T2
	var lastOK, firstFrozen time.Time
	for time.Now().Before(T2.Add(-500 * time.Millisecond)) {
		arena.mu.RLock()
		var tile uint32
		for t := range arena.byFlag["at"] {
			tile = t
			break
		}
		arena.mu.RUnlock()
		if tile == 0 {
			time.Sleep(300 * time.Millisecond)
			continue
		}
		r := players[2].click(ctx, g.N[tile][0], true)
		if r == "frozen" && firstFrozen.IsZero() {
			firstFrozen = time.Now()
		} else if r != "frozen" && firstFrozen.IsZero() {
			lastOK = time.Now()
		}
		time.Sleep(250 * time.Millisecond)
	}
	wg.Wait()
	log.check(!firstFrozen.IsZero() && T2.Sub(firstFrozen) > 7*time.Second && T2.Sub(firstFrozen) < 9*time.Second,
		"the end's freeze began %.1fs before T2 (last click let through %.1fs before)", T2.Sub(firstFrozen).Seconds(),
		T2.Sub(lastOK).Seconds())

	// The arena's map at its end, read past the cache inside the end's freeze, against the stream's.
	time.Sleep(time.Until(T2.Add(-1500 * time.Millisecond)))
	end := readMap()
	arena.mu.RLock()
	diffOwner, diffShield := 0, 0
	for tile := 1; tile <= siegeTiles; tile++ {
		if arena.m.owner[tile] != end.owner[tile] {
			diffOwner++
		}
		if arena.m.shields[tile] != end.shields[tile] {
			diffShield++
		}
	}
	applied, stale := arena.applied, arena.stale
	arenaOut := len(arena.outBy)
	arena.mu.RUnlock()
	log.check(diffOwner == 0 && diffShield == 0,
		"the stream told every change: the observer's map matches GetMap at the end (%d flags, %d shields differ; %d updates applied)",
		diffOwner, diffShield, applied)
	log.note("  %d updates older than the seed snapshot were skipped", stale)
	log.check(len(end.out) == arenaOut && arenaOut == len(knockOuts),
		"knock-outs: %d heard on the stream, %d in GetMap's knocked_out", len(knockOuts), len(end.out))
	endHeld := end.holdings()
	log.note("the arena at its end: %d tiles held, leaders %s; out: %s", sum(endHeld), topOf(endHeld, 8), strings.Join(end.out, ", "))
	sizes := map[int]int{}
	obsMu.Lock()
	for _, s := range spreadSizes {
		sizes[s]++
	}
	log.note("spreads heard: %d; tiles listed per spread: %v", len(spreadSizes), sizes)
	obsMu.Unlock()
	knockedTilesOK := true
	for _, k := range knockOuts {
		if end.owner[k.GetTileId()] == k.GetCountryId() {
			knockedTilesOK = false
		}
		for tile, o := range end.owner {
			if o == k.GetCountryId() {
				knockedTilesOK = false
				log.note("  %s is out but still holds tile %d", o, tile)
				break
			}
		}
	}
	log.check(knockedTilesOK, "every flag out holds nothing at the end")

	// ---- T2: the battle scored, the arena closed, the world's map back, day K+1 open
	var t2Cut time.Time
	select {
	case t2Cut = <-cutCh:
	case <-time.After(30 * time.Second):
	}
	log.check(!t2Cut.IsZero(), "the arena closed and cut every planet stream %.1fs after T2", t2Cut.Sub(T2).Seconds())
	var back siegeMap
	deadline = time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		back = readMap()
		if back.epoch != fresh.epoch {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	log.check(back.epoch != fresh.epoch, "GetMap answers the world's map again, a new epoch (%d)", back.epoch)
	time.Sleep(6 * time.Second)
	back = readMap()
	diffOwner, diffShield = 0, 0
	for tile := 1; tile <= siegeTiles; tile++ {
		if world.owner[tile] != back.owner[tile] {
			diffOwner++
		}
		if world.shields[tile] != back.shields[tile] {
			diffShield++
		}
	}
	log.check(diffOwner == 0 && diffShield == 0, "the world's map is back tile for tile: %d flags and %d shields differ of %d tiles",
		diffOwner, diffShield, siegeTiles)
	log.check(len(back.out) == 0, "no flag is out on the world's map")

	var battleRound *scoringv1.Round
	for range 50 {
		r, err := scoring.GetRound(ctx, connect.NewRequest(&scoringv1.GetRoundRequest{RoundId: battleID}))
		if err == nil && r.Msg.GetRound().GetEndedAtUnixMs() != 0 {
			battleRound = r.Msg.GetRound()
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if log.check(battleRound != nil, "scoring closed the battle's round") {
		log.check(math.Abs(float64(battleRound.GetEndedAtUnixMs()-T2.UnixMilli())) < 1000, "the battle ended at T2")
		takes := map[string]int{}
		for _, tk := range battleRound.GetTakes() {
			takes[tk.GetCountryId()] = int(tk.GetTiles())
		}
		// FightersRanking: the flags whose players took a tile, by the sum of their ranks on the tiles held at the
		// end and on the tiles taken, the takes' rank breaking a tie.
		fought := map[string]int{}
		for f := range takes {
			fought[f] = endHeld[f]
		}
		heldRank, takeRank := placesOf(fought), placesOf(takes)
		type place struct {
			f        string
			sum, tie int
		}
		var all []place
		for f := range fought {
			all = append(all, place{f, heldRank[f] + takeRank[f], takeRank[f]})
		}
		sort.Slice(all, func(i, j int) bool {
			if all[i].sum != all[j].sum {
				return all[i].sum < all[j].sum
			}
			return all[i].tie < all[j].tie
		})
		want := map[string]int{}
		for i, p := range all {
			if i > 0 && all[i-1].sum == p.sum && all[i-1].tie == p.tie {
				want[p.f] = want[all[i-1].f]
			} else {
				want[p.f] = i + 1
			}
		}
		bad := 0
		var table []string
		for _, s := range battleRound.GetStandings() {
			f := s.GetCountryId()
			table = append(table, fmt.Sprintf("%d. %s %dpts (held %d, took %d)", s.GetRank(), f, s.GetPoints(), endHeld[f], takes[f]))
			if int(s.GetRank()) != want[f] || s.GetPoints() != pointsAt(want[f], 2) {
				bad++
				log.note("  battle %s: rank %d points %d, expected rank %d points %d", f, s.GetRank(), s.GetPoints(), want[f], pointsAt(want[f], 2))
			}
		}
		log.note("battle standings: %s", strings.Join(table, "; "))
		log.check(bad == 0 && len(battleRound.GetStandings()) == len(want),
			"the battle placed the %d flags that fought, eliminated ones included, ×2 points (%d off)", len(want), bad)
		if dayRound != nil {
			expected := map[string]uint32{}
			for _, s := range dayRound.GetStandings() {
				expected[s.GetCountryId()] += s.GetPoints()
			}
			for _, s := range battleRound.GetStandings() {
				expected[s.GetCountryId()] += s.GetPoints()
			}
			scores, err := scoring.GetScores(ctx, connect.NewRequest(&scoringv1.GetScoresRequest{BookId: book}))
			must(t, err)
			bad = 0
			for _, s := range scores.Msg.GetScores().GetScores() {
				if s.GetPoints() != expected[s.GetCountryId()] {
					bad++
					log.note("  book %s: %d points, expected %d", s.GetCountryId(), s.GetPoints(), expected[s.GetCountryId()])
				}
			}
			log.check(bad == 0, "the season's table adds day K's and the battle's points (%d flags off)", bad)
		}
	}

	var afterRules bool
	for range 60 {
		pp, op, qp, ap = rulesOf()
		if pp == "default" && op == "default" && qp == "default" && ap == "default" {
			afterRules = true
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	log.check(afterRules, "every module plays default again: planet %s, offers %s, quizzes %s, antibot %s", pp, op, qp, ap)
	cr, err = public.GetChargeRules(ctx, connect.NewRequest(&planetv1.GetChargeRulesRequest{}))
	must(t, err)
	log.check(!cr.Msg.GetReach() && !cr.Msg.GetKnockOut() && !cr.Msg.GetSpreadUnlimited(), "GetChargeRules says the world's rules again")
	obsMu.Lock()
	sawRules = len(rulesSeen) > 0 && rulesSeen[len(rulesSeen)-1].After(t2Cut)
	obsMu.Unlock()
	log.check(sawRules, "the observer's stream heard rules_changed after the map came back")

	var next *gamemasterv1.GetTurnResponse
	for range 60 {
		nt, err := turns.GetTurn(ctx, connect.NewRequest(&gamemasterv1.GetTurnRequest{}))
		if err == nil && nt.Msg.GetTimeline().GetToday().GetRoundId() != dayK && nt.Msg.GetTimeline().GetToday().GetRoundId() != "" {
			next = nt.Msg
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	if log.check(next != nil, "the gamemaster tells day K+1") {
		day := next.GetTimeline().GetToday()
		_, err := scoring.GetRound(ctx, connect.NewRequest(&scoringv1.GetRoundRequest{RoundId: day.GetRoundId()}))
		log.check(err == nil && day.GetStartsAt().AsTime().Equal(T2) && next.GetTurn().GetKind() != gamemasterv1.EntryKind_ENTRY_KIND_BATTLE,
			"day K+1 (%s) starts at T2 on the world's map, its round open in scoring: %v", day.GetRoundId(), err)
	}
	r = (&siegePlayer{f: players[7].f, flag: players[7].flag}).click(ctx, g.C["au"], false)
	log.check(r == "ok", "after the battle a click lands anywhere again, Canberra too (%s)", r)

	// The hands.
	var lines []string
	for _, p := range players {
		var parts []string
		for k, v := range p.counts {
			parts = append(parts, fmt.Sprintf("%s %d", k, v))
		}
		sort.Strings(parts)
		lines = append(lines, fmt.Sprintf("%s(%s): %s", p.flag, p.role, strings.Join(parts, ", ")))
	}
	log.note("each hand's clicks:\n    %s", strings.Join(lines, "\n    "))
	obsMu.Lock()
	for _, c := range cuts {
		log.note("stream cut at %s: %s", c.at.UTC().Format("15:04:05.0"), c.err)
	}
	obsMu.Unlock()

	fmt.Printf("\n==== %d checks failed ====\n%s\n", len(log.failed), strings.Join(log.failed, "\n"))
	if out := os.Getenv("SIEGE_REPORT"); out != "" {
		_ = os.WriteFile(out, []byte(strings.Join(log.lines, "\n")+"\n"), 0o644)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func sum(m map[string]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}

func topOf(m map[string]int, n int) string {
	type kv struct {
		k string
		v int
	}
	var all []kv
	for k, v := range m {
		if v > 0 {
			all = append(all, kv{k, v})
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].v > all[j].v })
	var parts []string
	for i, e := range all {
		if i == n {
			break
		}
		parts = append(parts, fmt.Sprintf("%s %d", e.k, e.v))
	}
	return strings.Join(parts, ", ")
}
