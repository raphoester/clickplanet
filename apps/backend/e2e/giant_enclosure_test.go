package e2e_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1/antibotv1connect"
	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1/gamemasterv1connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1/progressionv1connect"
	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1/scoringv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
)

// A ring the in-process test computed from the real map (CP_GIANT_DUMP): the wall to hold, the tile that closes it,
// and every tile the closing click must fill.
type giantRing struct {
	Wall    []uint32 `json:"wall"`
	Closing uint32   `json:"closing"`
	Inside  []uint32 `json:"inside"`
}

var fast = &http.Client{Transport: &http.Transport{MaxIdleConnsPerHost: 512, MaxConnsPerHost: 512}}

type fighter struct {
	t       *testing.T
	stack   gameStack
	ip      string
	token   string
	cookie  string
	account string
	flag    string
}

func (s gameStack) fighter(t *testing.T, ip, flag string) *fighter {
	t.Helper()
	req := connect.NewRequest(&authv1.CreateSessionRequest{AttestationToken: "unused"})
	req.Header().Set("X-Real-IP", ip)
	res, err := authv1connect.NewAuthServiceClient(fast, s.baseURL).CreateSession(t.Context(), req)
	require.NoError(t, err)
	cookie, err := http.ParseSetCookie(res.Header().Get("Set-Cookie"))
	require.NoError(t, err)
	f := &fighter{t: t, stack: s, ip: ip, token: res.Msg.GetToken(), cookie: cookie.Name + "=" + cookie.Value, flag: flag}

	me := connect.NewRequest(&authv1.GetMeRequest{})
	f.send(me.Header())
	got, err := authv1connect.NewAuthServiceClient(fast, s.baseURL).GetMe(t.Context(), me)
	require.NoError(t, err)
	f.account = got.Msg.GetAccountId()

	pledge := connect.NewRequest(&planetv1.PledgeAllegianceRequest{CountryId: flag})
	f.send(pledge.Header())
	_, err = planetv1connect.NewClickServiceClient(fast, s.baseURL).PledgeAllegiance(t.Context(), pledge)
	require.NoError(t, err)
	return f
}

func (f *fighter) send(header http.Header) {
	header.Set("X-Real-IP", f.ip)
	header.Set(cpconnect.SessionHeader, f.token)
	header.Set("Cookie", f.cookie)
}

// click retries a throttled click, as a player waits for the bucket.
func (f *fighter) click(ctx context.Context, tile uint32, enclose bool) (time.Duration, error) {
	for {
		req := connect.NewRequest(&planetv1.ClickRequest{TileId: tile, CountryId: f.flag, Enclose: enclose})
		f.send(req.Header())
		start := time.Now()
		_, err := planetv1connect.NewClickServiceClient(fast, f.stack.baseURL).Click(ctx, req)
		took := time.Since(start)
		if connect.CodeOf(err) == connect.CodeResourceExhausted {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		return took, err
	}
}

func (f *fighter) tilesTaken() int64 {
	req := connect.NewRequest(&progressionv1.GetStatsRequest{})
	f.send(req.Header())
	res, err := progressionv1connect.NewProgressionServiceClient(fast, f.stack.baseURL).GetStats(f.t.Context(), req)
	if err != nil {
		return -1
	}
	return int64(res.Msg.GetStats().GetTilesTaken())
}

func (f *fighter) tally(book string) int64 {
	req := connect.NewRequest(&scoringv1.GetMyTallyRequest{BookId: book})
	f.send(req.Header())
	res, err := scoringv1connect.NewScoringServiceClient(fast, f.stack.baseURL).GetMyTally(f.t.Context(), req)
	if err != nil {
		return -1
	}
	return int64(res.Msg.GetTiles())
}

// paint clicks every tile, spread over the fighters, and returns each click's latency.
func paint(t *testing.T, fighters []*fighter, tiles []uint32, enclose bool) []time.Duration {
	t.Helper()
	var mu sync.Mutex
	var latencies []time.Duration
	var next atomic.Int64
	var wg sync.WaitGroup
	for _, f := range fighters {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				i := int(next.Add(1) - 1)
				if i >= len(tiles) {
					return
				}
				took, err := f.click(t.Context(), tiles[i], enclose)
				if err != nil {
					t.Errorf("click on %d for %s: %v", tiles[i], f.flag, err)
					return
				}
				mu.Lock()
				latencies = append(latencies, took)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return latencies
}

func percentiles(latencies []time.Duration) string {
	if len(latencies) == 0 {
		return "none"
	}
	sorted := slices.Clone(latencies)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	at := func(p float64) time.Duration {
		return sorted[min(len(sorted)-1, int(float64(len(sorted))*p))].Round(time.Microsecond)
	}
	return fmt.Sprintf("n=%d p50=%s p95=%s p99=%s max=%s", len(sorted), at(0.5), at(0.95), at(0.99), sorted[len(sorted)-1].Round(time.Microsecond))
}

// A planet stream, read as the page reads it.
type listener struct {
	enclosures []*planetv1.TilesEnclosed
	heardAt    []time.Time
	fortified  int
	updates    int
	frames     int
	ended      error
	open       bool
	mu         sync.Mutex
}

func (s gameStack) listen(t *testing.T, ctx context.Context) *listener {
	t.Helper()
	stream, err := planetv1connect.NewClickServiceClient(fast, s.baseURL).
		ListenForEvents(ctx, connect.NewRequest(&planetv1.ListenForEventsRequest{}))
	require.NoError(t, err)
	require.True(t, stream.Receive(), "the stream opens with a heartbeat")
	l := &listener{open: true}
	go func() {
		for stream.Receive() {
			msg := stream.Msg()
			l.mu.Lock()
			l.frames++
			switch {
			case msg.GetTilesEnclosed() != nil:
				l.enclosures = append(l.enclosures, msg.GetTilesEnclosed())
				l.heardAt = append(l.heardAt, time.Now())
			case msg.GetLandmassFortified() != nil:
				l.fortified++
			case msg.GetTileUpdate() != nil:
				l.updates++
			}
			l.mu.Unlock()
		}
		l.mu.Lock()
		l.ended, l.open = stream.Err(), false
		l.mu.Unlock()
	}()
	return l
}

func (l *listener) enclosureAt(tile uint32) *planetv1.TilesEnclosed {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, e := range l.enclosures {
		if e.GetClosingTileId() == tile {
			return e
		}
	}
	return nil
}

func (l *listener) heard(tile uint32) time.Time {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, e := range l.enclosures {
		if e.GetClosingTileId() == tile {
			return l.heardAt[i]
		}
	}
	return time.Time{}
}

type groupLag struct {
	name    string
	lag     int64
	pending int64
	read    int64
}

func takeGroups(t *testing.T, rdb *redis.Client) []groupLag {
	t.Helper()
	groups, err := rdb.XInfoGroups(t.Context(), "events:planet.v1.TilesTaken").Result()
	require.NoError(t, err)
	lags := make([]groupLag, 0, len(groups))
	for _, g := range groups {
		lags = append(lags, groupLag{name: g.Name, lag: g.Lag, pending: g.Pending, read: g.EntriesRead})
	}
	return lags
}

func usedMemory(t *testing.T, rdb *redis.Client) string {
	t.Helper()
	info, err := rdb.Info(t.Context(), "memory").Result()
	require.NoError(t, err)
	for _, line := range strings.Split(info, "\n") {
		if strings.HasPrefix(line, "used_memory_human:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "used_memory_human:"))
		}
	}
	return "?"
}

// waitForTheTakes polls each TilesTaken group until none lags: how long each module takes to hear a giant pocket.
func waitForTheTakes(t *testing.T, rdb *redis.Client, since time.Time, within time.Duration) map[string]time.Duration {
	t.Helper()
	caughtUp := map[string]time.Duration{}
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		lags := takeGroups(t, rdb)
		all := true
		for _, g := range lags {
			if g.lag == 0 && g.pending == 0 {
				if _, seen := caughtUp[g.name]; !seen {
					caughtUp[g.name] = time.Since(since).Round(100 * time.Millisecond)
				}
			} else {
				all = false
			}
		}
		if all && len(lags) > 0 {
			return caughtUp
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Errorf("the take groups did not catch up within %s: %+v", within, takeGroups(t, rdb))
	return caughtUp
}

func loadRing(t *testing.T, dir, name string) giantRing {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, name+".json")) //nolint:gosec // the dir the one running it names.
	require.NoError(t, err)
	var r giantRing
	require.NoError(t, json.Unmarshal(raw, &r))
	return r
}

// Tonight's battle, Ring War, played on the whole stack: run by hand with CP_GIANT_SCENARIOS=<dir of the rings>.
func TestGiantEnclosuresOnTheWholeStack(t *testing.T) {
	dir := os.Getenv("CP_GIANT_SCENARIOS")
	if dir == "" {
		t.Skip("a stress run, by hand: CP_GIANT_SCENARIOS=<the dir CP_GIANT_DUMP wrote>")
	}
	russia, africa := loadRing(t, dir, "russia"), loadRing(t, dir, "africa")

	game := startGame(t)
	ctx := t.Context()
	rdb := redis.NewClient(&redis.Options{Addr: game.redis.Address, Password: game.redis.Password})
	t.Cleanup(func() { _ = rdb.Close() })

	// The set production plays tonight, played the way production plays it: a battle entry of the gamemaster.
	planetAdmin := planetv1connect.NewAdminServiceClient(fast, game.consoleURL)
	antibotAdmin := antibotv1connect.NewAdminServiceClient(fast, game.consoleURL)
	schedule := gamemasterv1connect.NewAdminServiceClient(fast, game.consoleURL)
	_, err := planetAdmin.SaveRuleSet(ctx, connect.NewRequest(&planetv1.SaveRuleSetRequest{Set: &planetv1.RuleSet{
		Name: "unlimited-enclose",
		Rules: &planetv1.Rules{
			Clicks: &planetv1.ClickRules{Tempo: proto.Float64(3)},
			Bonuses: &planetv1.BonusRules{
				Kinds:   &planetv1.KindWeights{EncloseClicks: proto.Float64(0)},
				Enclose: &planetv1.EncloseRules{MaxTiles: proto.Uint32(1_000_000), Unlimited: proto.Bool(true)},
			},
			AllegianceLocked: proto.Bool(true),
		},
	}}))
	require.NoError(t, err)
	antibotRules, err := antibotAdmin.GetRules(ctx, connect.NewRequest(&antibotv1.GetRulesRequest{}))
	require.NoError(t, err)
	_, err = antibotAdmin.SaveRuleSet(ctx, connect.NewRequest(&antibotv1.SaveRuleSetRequest{
		Set: &antibotv1.RuleSet{Name: "unlimited-enclose", Rules: antibotRules.Msg.GetSets()[0].GetRules()},
	}))
	require.NoError(t, err)

	// Every fighter pledges before the battle locks the flags.
	ru, ua, fr, br := make([]*fighter, 0, 16), make([]*fighter, 0, 8), make([]*fighter, 0, 8), make([]*fighter, 0, 6)
	for i := range 16 {
		ru = append(ru, game.fighter(t, fmt.Sprintf("198.51.100.%d", 10+i), "ru"))
	}
	for i := range 8 {
		ua = append(ua, game.fighter(t, fmt.Sprintf("198.51.100.%d", 40+i), "ua"))
		fr = append(fr, game.fighter(t, fmt.Sprintf("198.51.100.%d", 60+i), "fr"))
	}
	for i := range 3 {
		br = append(br, game.fighter(t, fmt.Sprintf("198.51.100.%d", 80+i), "br"))
		br = append(br, game.fighter(t, fmt.Sprintf("198.51.100.%d", 90+i), "ar"))
	}
	ringer := ua[0]
	africaRinger := fr[0]

	// The season's book, read while today's round still plays: the battle's round replaces it.
	book := game.today(t).book
	startsAt := time.Now().Add(2 * time.Second)
	endsAt := startsAt.UTC().Truncate(24 * time.Hour).Add(21 * time.Hour)
	if !endsAt.After(startsAt) {
		endsAt = endsAt.Add(24 * time.Hour)
	}
	added, err := schedule.AddEntry(ctx, connect.NewRequest(&gamemasterv1.AddEntryRequest{
		RuleSet: "unlimited-enclose", Kind: gamemasterv1.EntryKind_ENTRY_KIND_BATTLE,
		Description: "Ring War: close a ring around any land, and one click takes everything inside.",
		StartsAt:    timestamppb.New(startsAt), EndsAt: timestamppb.New(endsAt), Priority: 1, Times: 2,
	}))
	require.NoError(t, err)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		rules, err := planetAdmin.GetRules(ctx, connect.NewRequest(&planetv1.GetRulesRequest{}))
		require.NoError(c, err)
		assert.Equal(c, "unlimited-enclose", rules.Msg.GetPlaying())
	}, 20*time.Second, 100*time.Millisecond, "the gamemaster commands the battle's set to planet")
	bonusRules, err := planetv1connect.NewClickServiceClient(fast, game.baseURL).
		GetBonusRules(ctx, connect.NewRequest(&planetv1.GetBonusRulesRequest{}))
	require.NoError(t, err)
	require.True(t, bonusRules.Msg.GetEnclosuresUnlimited(), "the client is told enclosures are free")
	t.Logf("battle %s plays: enclose max_tiles=%d unlimited=%v", added.Msg.GetId(), bonusRules.Msg.GetEnclosureMaxTiles(),
		bonusRules.Msg.GetEnclosuresUnlimited())

	// Russia: its own flag holds the inside; Ukraine builds the ring, with enclose on as the page sends every click tonight.
	began := time.Now()
	ruPaint := paint(t, ru, russia.Inside, false)
	t.Logf("ru painted the inside of russia (%d tiles) in %s: %s", len(russia.Inside), time.Since(began).Round(time.Millisecond), percentiles(ruPaint))
	began = time.Now()
	uaWall := paint(t, ua, russia.Wall, true)
	t.Logf("ua built russia's ring (%d tiles, enclose on) in %s: %s", len(russia.Wall), time.Since(began).Round(time.Millisecond), percentiles(uaWall))
	waitForTheTakes(t, rdb, time.Now(), 5*time.Minute)

	closeRing(t, game, rdb, "russia", russia, ringer, br, book, "ru")

	// Africa, blank inside, right after: France rings the whole continent.
	began = time.Now()
	frWall := paint(t, fr, africa.Wall, true)
	t.Logf("fr built africa's ring (%d tiles, enclose on) in %s: %s", len(africa.Wall), time.Since(began).Round(time.Millisecond), percentiles(frWall))
	waitForTheTakes(t, rdb, time.Now(), 5*time.Minute)

	closeRing(t, game, rdb, "africa", africa, africaRinger, br, book, "")

	dead, err := rdb.XLen(ctx, "events:dead").Result()
	require.NoError(t, err)
	assert.Zero(t, dead, "no subscriber gave up on an event")
	t.Logf("redis after both: %s used; TilesTaken stream holds %d", usedMemory(t, rdb), rdb.XLen(ctx, "events:planet.v1.TilesTaken").Val())
}

func closeRing(t *testing.T, game gameStack, rdb *redis.Client, name string, ring giantRing, ringer *fighter, busy []*fighter,
	book string, previous string,
) {
	t.Helper()
	ctx := t.Context()

	// The pages watching: 25 streams.
	listening, stopListening := context.WithCancel(ctx)
	defer stopListening()
	pages := make([]*listener, 0, 25)
	for range 25 {
		pages = append(pages, game.listen(t, listening))
	}

	// The battle goes on elsewhere: two flags fight over the same tiles, ten clicks a second each, enclose on.
	elsewhere := []uint32{}
	for tile := uint32(200000); len(elsewhere) < 3000; tile += 7 {
		elsewhere = append(elsewhere, tile)
	}
	type sample struct {
		at   time.Time
		took time.Duration
	}
	stop := make(chan struct{})
	var mu sync.Mutex
	var samples []sample
	var wg sync.WaitGroup
	for i, f := range busy {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tick := time.NewTicker(100 * time.Millisecond)
			defer tick.Stop()
			for n := i / 2; ; n++ {
				select {
				case <-stop:
					return
				case <-tick.C:
				}
				began := time.Now()
				took, err := f.click(ctx, elsewhere[n%len(elsewhere)], true)
				if err != nil && !errors.Is(err, context.Canceled) {
					t.Errorf("a busy click: %v", err)
					return
				}
				mu.Lock()
				samples = append(samples, sample{at: began, took: took})
				mu.Unlock()
			}
		}()
	}
	window := func(from, to time.Time) []time.Duration {
		mu.Lock()
		defer mu.Unlock()
		var in []time.Duration
		for _, s := range samples {
			if !s.at.Before(from) && s.at.Before(to) {
				in = append(in, s.took)
			}
		}
		return in
	}
	defer func() {
		close(stop)
		wg.Wait()
	}()
	time.Sleep(3 * time.Second)

	closedAt := time.Now()
	took, err := ringer.click(ctx, ring.Closing, true)
	require.NoError(t, err)
	t.Logf("%s: the closing click answered in %s", name, took.Round(time.Microsecond))
	time.Sleep(2 * time.Second)
	t.Logf("%s: others' clicks, 3s before the ring closed: %s", name, percentiles(window(closedAt.Add(-3*time.Second), closedAt)))
	t.Logf("%s: others' clicks, 2s after the ring closed:  %s", name, percentiles(window(closedAt, closedAt.Add(2*time.Second))))

	// Every page hears one enclosure with every tile inside, and stays open.
	want := slices.Clone(ring.Inside)
	slices.Sort(want)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		for _, page := range pages {
			assert.NotNil(c, page.enclosureAt(ring.Closing))
		}
	}, 10*time.Second, 20*time.Millisecond, "every page hears the enclosure")
	var heardIn time.Duration
	for _, page := range pages {
		heardIn = max(heardIn, page.heard(ring.Closing).Sub(closedAt))
	}
	for i, page := range pages {
		got := page.enclosureAt(ring.Closing)
		filled := slices.Clone(got.GetFilledTileIds())
		slices.Sort(filled)
		assert.Len(t, filled, len(want), "page %d hears every tile inside", i)
		assert.Equal(t, want, filled, "page %d hears exactly the inside", i)
		assert.Equal(t, ringer.flag, got.GetCountryId())
		assert.Empty(t, got.GetStruck())
		page.mu.Lock()
		assert.True(t, page.open, "page %d's stream was not cut: %v", i, page.ended)
		page.mu.Unlock()
	}
	frame, err := proto.Marshal(&planetv1.PlanetEvent{Event: &planetv1.PlanetEvent_TilesEnclosed{TilesEnclosed: pages[0].enclosureAt(ring.Closing)}})
	require.NoError(t, err)
	pages[0].mu.Lock()
	fortified := pages[0].fortified
	pages[0].mu.Unlock()
	t.Logf("%s: the last of 25 pages heard the enclosure %s after the click was sent: %d tiles in one %d KB frame, %d landmasses fortified after it",
		name, heardIn.Round(time.Millisecond), len(want), len(frame)/1024, fortified)

	// The map every newcomer loads, and who holds a tile.
	clicks := planetv1connect.NewClickServiceClient(fast, game.baseURL, connect.WithHTTPGet())
	inside := ring.Inside[len(ring.Inside)/2]
	asked := time.Now()
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		holder, err := clicks.GetTileHolder(ctx, connect.NewRequest(&planetv1.GetTileHolderRequest{TileId: inside}))
		require.NoError(c, err)
		assert.Equal(c, ringer.account, holder.Msg.GetHolder().GetAccountId())
	}, 10*time.Second, 50*time.Millisecond, "a tile inside is held by the ringer, once the ledger is flushed")
	t.Logf("%s: GetTileHolder names the ringer %s after the frame", name, time.Since(asked).Round(time.Millisecond))
	holderCalls := make([]time.Duration, 0, 40)
	for i := range 40 {
		tile := ring.Inside[(i*7919)%len(ring.Inside)]
		if i%2 == 1 {
			tile = elsewhere[i*37%len(elsewhere)]
		}
		began := time.Now()
		_, err := clicks.GetTileHolder(ctx, connect.NewRequest(&planetv1.GetTileHolderRequest{TileId: tile}))
		require.NoError(t, err)
		holderCalls = append(holderCalls, time.Since(began))
	}
	t.Logf("%s: GetTileHolder, inside and elsewhere: %s", name, percentiles(holderCalls))
	for _, probe := range []struct {
		what string
		tile uint32
	}{{"inside", inside}, {"a tile taken by clicks", elsewhere[1]}, {"a tile nobody took", 3}} {
		rows, err := game.schema(t, "planet").On(ctx).QueryContext(ctx, "EXPLAIN (ANALYZE, BUFFERS) "+holderSQL, probe.tile)
		require.NoError(t, err)
		var plan []string
		for rows.Next() {
			var line string
			require.NoError(t, rows.Scan(&line))
			if strings.Contains(line, "Scan") || strings.Contains(line, "Execution") || strings.Contains(line, "Buffers") {
				plan = append(plan, strings.TrimSpace(line))
			}
		}
		require.NoError(t, rows.Close())
		t.Logf("%s: GetTileHolder's plan for %s:\n    %s", name, probe.what, strings.Join(plan, "\n    "))
	}
	mapped, err := clicks.GetMap(ctx, connect.NewRequest(&planetv1.GetMapRequest{}))
	require.NoError(t, err)
	owned := 0
	codes := mapped.Msg.GetCodes()
	tiles := mapped.Msg.GetTiles()
	start := mapped.Msg.GetStartTileId()
	for _, tile := range ring.Inside {
		at := int(tile-start) * 2
		if at+1 < len(tiles) && codes[int(tiles[at])|int(tiles[at+1])<<8] == ringer.flag {
			owned++
		}
	}
	assert.Equal(t, len(ring.Inside), owned, "GetMap shows every tile inside as the ringer's")

	// Postgres: the tiles flushed, one enclose in the ledger, and the relay past it.
	planetDB := game.schema(t, "planet")
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		var flushed int
		require.NoError(c, planetDB.On(ctx).QueryRowContext(ctx,
			`SELECT count(*) FROM tiles WHERE id = ANY($1) AND country = $2`, pq.Array(int64s(ring.Inside)), ringer.flag).Scan(&flushed))
		assert.Equal(c, len(ring.Inside), flushed)
	}, 10*time.Second, 100*time.Millisecond, "the flush writes every tile inside to postgres")
	var position int64
	var taken, payloadBytes int
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		require.NoError(c, planetDB.On(ctx).QueryRowContext(ctx, `
			SELECT position, jsonb_array_length(payload::jsonb -> 'taken'), length(payload::text) FROM ledger_events
			WHERE kind = 'enclose' AND tile = $1 ORDER BY position DESC LIMIT 1`, int64(ring.Closing)).Scan(&position, &taken, &payloadBytes))
	}, 10*time.Second, 100*time.Millisecond, "the ledger keeps the enclosure")
	assert.Equal(t, len(ring.Inside), taken, "the ledger's enclose names every tile taken")
	var takenFrom int
	if previous != "" {
		require.NoError(t, planetDB.On(ctx).QueryRowContext(ctx, `
			SELECT count(*) FROM ledger_events, jsonb_array_elements(payload::jsonb -> 'taken') AS t
			WHERE position = $1 AND t ->> 'owner' = $2`, position, previous).Scan(&takenFrom))
		assert.Equal(t, len(ring.Inside), takenFrom, "each tile is written down with the flag it was taken from")
	}
	t.Logf("%s: ledger row at %d holds %d tiles, %d KB of payload", name, position, taken, payloadBytes/1024)

	// Redis and every module behind it.
	var streamLen int64
	require.Eventually(t, func() bool {
		var relayed int64
		_ = planetDB.On(ctx).QueryRowContext(ctx, `SELECT position FROM ledger_relayed`).Scan(&relayed)
		return relayed > position
	}, 20*time.Second, 100*time.Millisecond, "the relay tells the enclosure to Redis")
	streamLen = rdb.XLen(ctx, "events:planet.v1.TilesTaken").Val()
	t.Logf("%s: relayed; TilesTaken stream holds %d entries, redis uses %s", name, streamLen, usedMemory(t, rdb))
	caughtUp := waitForTheTakes(t, rdb, closedAt, 30*time.Minute)
	t.Logf("%s: others' clicks while the modules took the pocket in: %s", name, percentiles(window(closedAt.Add(2*time.Second), time.Now())))
	names := make([]string, 0, len(caughtUp))
	for group := range caughtUp {
		names = append(names, group)
	}
	sort.Strings(names)
	for _, group := range names {
		t.Logf("%s: group %-28s caught up %s after the click", name, group, caughtUp[group])
	}
	t.Logf("%s: redis uses %s once every group caught up", name, usedMemory(t, rdb))

	// What the ledger says the ringer took, every click and every tile of its spreads and enclosures, once all of it is
	// flushed and told: progression and scoring must count exactly that, no tile missed and none twice.
	var wantTaken int64
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		var head, relayed int64
		require.NoError(c, planetDB.On(ctx).QueryRowContext(ctx, `SELECT COALESCE(max(position) + 1, 0) FROM ledger_events`).Scan(&head))
		require.NoError(c, planetDB.On(ctx).QueryRowContext(ctx, `SELECT position FROM ledger_relayed`).Scan(&relayed))
		assert.GreaterOrEqual(c, relayed, head)
	}, 20*time.Second, 200*time.Millisecond, "the relay told everything the ledger flushed")
	waitForTheTakes(t, rdb, time.Now(), 5*time.Minute)
	require.NoError(t, planetDB.On(ctx).QueryRowContext(ctx, `
		SELECT (SELECT count(*) FROM ledger_events WHERE account = $1 AND kind = 'take')
			+ (SELECT COALESCE(sum(jsonb_array_length(payload::jsonb -> 'taken')), 0) FROM ledger_events
			   WHERE account = $1 AND kind IN ('spread', 'enclose'))
	`, ringer.account).Scan(&wantTaken))
	require.Eventually(t, func() bool { return ringer.tilesTaken() == wantTaken }, 30*time.Second, 200*time.Millisecond,
		"progression counts every tile the ledger says the ringer took, once: has %d, wants %d", ringer.tilesTaken(), wantTaken)
	require.Eventually(t, func() bool { return ringer.tally(book) == wantTaken }, 30*time.Second, 200*time.Millisecond,
		"scoring counts every tile the ledger says the ringer took, once: has %d, wants %d", ringer.tally(book), wantTaken)
	t.Logf("%s: progression and scoring each count %d tiles for the ringer", name, wantTaken)
}

// holder_query's lastChange, copied: e2e cannot import planet's internals.
const holderSQL = `
	SELECT COALESCE(account::text, ''), country, taken_at FROM (
		(SELECT position, account, country, taken_at FROM ledger_events
		WHERE kind = 'take' AND tile = $1
		ORDER BY position DESC LIMIT 1)
		UNION ALL
		(SELECT position, account, country, taken_at FROM ledger_events
		WHERE kind IN ('spread', 'enclose')
			AND payload @> jsonb_build_object('taken', jsonb_build_array(jsonb_build_object('tile', $1::integer)))
		ORDER BY position DESC LIMIT 1)
		UNION ALL
		(SELECT position, account, '', taken_at FROM ledger_events
		WHERE kind = 'bomb'
			AND payload @> jsonb_build_object('cleared', jsonb_build_array(jsonb_build_object('tile', $1::integer)))
		ORDER BY position DESC LIMIT 1)
		ORDER BY position DESC LIMIT 1
	) last
	WHERE country <> ''
`

func int64s(tiles []uint32) []int64 {
	out := make([]int64, len(tiles))
	for i, tile := range tiles {
		out[i] = int64(tile)
	}
	return out
}
