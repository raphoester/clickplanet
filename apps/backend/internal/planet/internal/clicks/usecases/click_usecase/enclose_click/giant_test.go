package enclose_click_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/embedded_geodesic_map"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/enclose_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/relay_ledger_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/planetmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const mapTiles = 260503

type world struct {
	geography *clicks.Geography
	borders   *clicks.Borders
}

var loadWorld = sync.OnceValues(func() (world, error) {
	loader := embedded_geodesic_map.New(mapTiles, nil)
	geography, err := loader.LoadGeography()
	if err != nil {
		return world{}, fmt.Errorf("failed to load the geography: %w", err)
	}
	borders, err := loader.LoadBorders()
	if err != nil {
		return world{}, fmt.Errorf("failed to load the borders: %w", err)
	}
	return world{geography: geography, borders: borders}, nil
})

// The ring a flag must hold to close the region holding seed: each tile of it that touches an edge or other ground.
type ring struct {
	wall    []uint32
	inside  []uint32 // every tile the closing click must fill
	closing uint32
}

func (w world) ringAround(region func(uint32) bool, seed uint32) ring {
	component := w.flood(seed, region)
	walled := cpcolls.NewSet[uint32]()
	for _, tile := range component {
		around := w.geography.Neighbours(tile)
		edge := len(around) < clicks.MaxDegree
		for _, next := range around {
			edge = edge || !region(next)
		}
		if edge {
			walled.Add(tile)
		}
	}

	// The biggest piece of the inside, and a wall tile beside it to close it with.
	var biggest []uint32
	seen := cpcolls.NewSet[uint32]()
	for _, tile := range component {
		if walled.Contains(tile) || seen.Contains(tile) {
			continue
		}
		piece := w.flood(tile, func(t uint32) bool { return region(t) && !walled.Contains(t) })
		seen.Add(piece...)
		if len(piece) > len(biggest) {
			biggest = piece
		}
	}

	var closing uint32
	for _, tile := range biggest {
		for _, next := range w.geography.Neighbours(tile) {
			if walled.Contains(next) {
				closing = next
				break
			}
		}
		if closing != 0 {
			break
		}
	}

	// What the click fills: every piece of the inside beside the closing tile.
	filled := cpcolls.NewSet[uint32]()
	for _, next := range w.geography.Neighbours(closing) {
		if walled.Contains(next) || filled.Contains(next) || !region(next) {
			continue
		}
		filled.Add(w.flood(next, func(t uint32) bool { return region(t) && !walled.Contains(t) })...)
	}

	r := ring{closing: closing}
	walled.ForEach(func(tile uint32) {
		if tile != closing {
			r.wall = append(r.wall, tile)
		}
	})
	filled.ForEach(func(tile uint32) { r.inside = append(r.inside, tile) })
	slices.Sort(r.wall)
	slices.Sort(r.inside)
	return r
}

func (w world) flood(seed uint32, keep func(uint32) bool) []uint32 {
	seen := cpcolls.NewSet(seed)
	queue := []uint32{seed}
	for i := 0; i < len(queue); i++ {
		for _, next := range w.geography.Neighbours(queue[i]) {
			if !seen.Contains(next) && keep(next) {
				seen.Add(next)
				queue = append(queue, next)
			}
		}
	}
	return queue
}

func (w world) groundOf(codes ...string) func(uint32) bool {
	return func(tile uint32) bool { return slices.Contains(codes, w.borders.CountryOf(tile)) }
}

func (w world) firstTileOf(code string) uint32 {
	best, size := uint32(0), 0
	for tile := uint32(1); tile <= mapTiles; tile++ {
		if w.borders.CountryOf(tile) != code {
			continue
		}
		if n := len(w.borders.TilesOf(w.borders.LandmassOf(tile))); n > size {
			best, size = tile, n
		}
	}
	return best
}

var africa = []string{
	"dz", "ao", "bj", "bw", "bf", "bi", "cm", "cv", "cf", "td", "km", "cd", "cg", "ci", "dj", "eg", "gq", "er", "sz",
	"et", "ga", "gm", "gh", "gn", "gw", "ke", "ls", "lr", "ly", "mg", "mw", "ml", "mr", "mu", "ma", "mz", "na", "ne",
	"ng", "rw", "st", "sn", "sc", "sl", "so", "za", "ss", "sd", "tz", "tg", "tn", "ug", "zm", "zw", "eh",
}

// The game as tonight's battle plays it: unlimited enclosures, up to a million tiles.
type game struct {
	world   world
	tiles   *inmemory_tile_storage.Storage
	ledger  *inmemory_ledger_storage.Storage
	click   click_usecase.IUseCase
	terrain bonuses.Terrain
	rules   bonuses.Rules
}

func newGame(t testing.TB, w world) *game {
	t.Helper()

	tiles := inmemory_tile_storage.New(w.borders, inmemory_tile_storage.Config{SubscriberBuffer: 1024},
		inmemory_tile_storage.NewMemoryPersistence(nil), cpatomic.FakeTransactioner{}, &inmemory_tile_storage.RecordedPlacements{}, cptime.SystemClock{}, nil)
	require.NoError(t, tiles.Load(t.Context(), clicks.World))
	takings := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence(), nil)
	require.NoError(t, takings.Load(t.Context()))

	rules := bonuses.Rules{Enclose: bonuses.EncloseRules{MaxTiles: 1_000_000, Unlimited: true}}
	writer := ledger.NewRecording(tiles, clicks.NewClaiming(tiles, func() int { return 10 }), takings, cptime.SystemClock{})
	terrain := bonuses.NewTerrain(w.geography, tiles)
	var chain click_usecase.IUseCase = click_usecase.New(clicks.NewBoard(mapTiles), writer, cpcountries.New())
	chain = enclose_click.New(chain, nil, terrain, enclose_click.NewAnnexer(writer, nil), func() bonuses.Rules { return rules })

	return &game{world: w, tiles: tiles, ledger: takings, click: chain, terrain: terrain, rules: rules}
}

func player(account, ip string) context.Context {
	return cpctx.AddAccountToContext(cpctx.AddIPToContext(context.Background(), ip), account)
}

// Paints without a click: the board as the battle found it.
func (g *game) paint(t testing.TB, tiles []uint32, flag string) {
	t.Helper()
	ctx := player("00000000-0000-4000-8000-000000000001", "198.51.100.1")
	for _, tile := range tiles {
		require.NoError(t, g.tiles.Set(ctx, tile, flag))
	}
}

// A stream, read as fast as a client reads it: what it was told, and whether it was cut.
type stream struct {
	enclosures []clicks.Enclosure
	versions   []clicks.MapVersion
	fortified  int
	updates    int
	cut        bool
	done       chan struct{}
}

func (g *game) listen(t testing.TB) (*stream, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	changes, _, err := g.tiles.Subscribe(ctx)
	require.NoError(t, err)
	s := &stream{done: make(chan struct{})}
	go func() {
		defer close(s.done)
		for {
			select {
			case change, open := <-changes:
				if !open {
					s.cut = ctx.Err() == nil
					return
				}
				switch {
				case change.Enclosure != nil:
					s.enclosures = append(s.enclosures, *change.Enclosure)
					s.versions = append(s.versions, change.Version)
				case change.Fortification != nil:
					s.fortified++
				case change.Update != nil:
					s.updates++
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return s, cancel
}

type report struct {
	name                 string
	wall, inside         int
	clickLatency         time.Duration
	search               time.Duration
	filled, struck, fort int
	frameBytes           int
	frameMarshal         time.Duration
	ledgerBytes          int
	facts                int
	factBytes            int
	relay                time.Duration
	worstOtherClick      time.Duration
	otherClicks          int64
}

func (r report) String() string {
	return fmt.Sprintf("%-26s wall=%-6d inside=%-7d click=%-9s (search %-9s) filled=%-7d struck=%-5d fortified=%-3d "+
		"frame=%-7s marshal=%-8s ledgerRow=%-7s facts=%-7d factBytes=%-8s relay=%-9s concurrentClicks=%d worst=%s",
		r.name, r.wall, r.inside, r.clickLatency.Round(time.Microsecond), r.search.Round(time.Microsecond), r.filled, r.struck,
		r.fort, kb(r.frameBytes), r.frameMarshal.Round(time.Microsecond), kb(r.ledgerBytes), r.facts, kb(r.factBytes),
		r.relay.Round(time.Millisecond), r.otherClicks, r.worstOtherClick.Round(time.Microsecond))
}

func kb(n int) string { return fmt.Sprintf("%.0fKB", float64(n)/1024) }

type countingSender struct {
	at       uint64
	messages int
	bytes    int
	tiles    map[string]int
}

type positioned interface{ GetPosition() uint64 }

func (s *countingSender) Send(_ context.Context, events ...proto.Message) error {
	for _, event := range events {
		if p, ok := event.(positioned); ok && p.GetPosition() != s.at {
			continue
		}
		payload, err := proto.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal a fact: %w", err)
		}
		s.messages++
		s.bytes += len(payload)
		switch told := event.(type) {
		case *planetv1.TilesTaken:
			s.tiles["taken"] += len(told.GetTiles())
		case *planetv1.ShieldsStruck:
			s.tiles["struck"] += len(told.GetTiles())
		}
	}
	return nil
}

type memoryCursor struct{ at ledger.Position }

func (c *memoryCursor) Relayed(context.Context) (ledger.Position, error) { return c.at, nil }
func (c *memoryCursor) SaveRelayed(_ context.Context, at ledger.Position) error {
	c.at = at
	return nil
}

// close clicks the last tile of the ring with enclose on, while other players click elsewhere on the map.
func (g *game) close(t testing.TB, name string, r ring, attacker string, busy bool) report {
	t.Helper()
	rep := report{name: name, wall: len(r.wall) + 1, inside: len(r.inside)}

	// The search alone, as the use case runs it after the take.
	g.paint(t, []uint32{r.closing}, attacker)
	start := time.Now()
	pocket := g.terrain.PocketClosedBy(r.closing, attacker, g.rules.Enclose.MaxTiles)
	rep.search = time.Since(start)
	require.Len(t, pocket.Inside(), len(r.inside), "the search finds every tile inside the ring")
	g.paint(t, []uint32{r.closing}, "")
	heard, stopListening := g.listen(t)

	// Other players clicking far away while the pocket is taken: how long the tiles lock makes them wait.
	stop := make(chan struct{})
	var wg sync.WaitGroup
	var worst atomic.Int64
	var others atomic.Int64
	if busy {
		far := g.world.firstTileOf("br")
		elsewhere := g.world.flood(far, g.world.groundOf("br"))[:2000]
		for i := range 4 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx := player(fmt.Sprintf("00000000-0000-4000-8000-00000000010%d", i), "198.51.100.9")
				for n := 0; ; n++ {
					select {
					case <-stop:
						return
					default:
					}
					began := time.Now()
					_, err := g.click.Execute(ctx, click_usecase.In{TileID: elsewhere[(n*4+i)%len(elsewhere)], CountryID: "br", Enclose: true})
					if err != nil {
						t.Error(err)
						return
					}
					took := time.Since(began)
					others.Add(1)
					for {
						old := worst.Load()
						if int64(took) <= old || worst.CompareAndSwap(old, int64(took)) {
							break
						}
					}
				}
			}()
		}
		time.Sleep(50 * time.Millisecond)
	}

	ctx := player("00000000-0000-4000-8000-0000000000aa", "198.51.100.2")
	start = time.Now()
	out, err := g.click.Execute(ctx, click_usecase.In{TileID: r.closing, CountryID: attacker, Enclose: true})
	rep.clickLatency = time.Since(start)
	require.NoError(t, err)
	require.Equal(t, clicks.Taken, out.Outcome)

	if busy {
		time.Sleep(50 * time.Millisecond)
		close(stop)
		wg.Wait()
		rep.worstOtherClick = time.Duration(worst.Load())
		rep.otherClicks = others.Load()
	}

	// The stream: one enclosure, with every tile inside.
	time.Sleep(100 * time.Millisecond)
	stopListening()
	<-heard.done
	require.False(t, heard.cut, "the stream was cut: the giant pocket overflowed the subscriber buffer")
	var enclosure *clicks.Enclosure
	var version clicks.MapVersion
	for i := range heard.enclosures {
		if heard.enclosures[i].ClosingTile == r.closing {
			enclosure, version = &heard.enclosures[i], heard.versions[i]
		}
	}
	require.NotNil(t, enclosure, "no enclosure on the stream")
	rep.fort = heard.fortified
	rep.filled, rep.struck = len(enclosure.Filled), len(enclosure.Struck)
	filled := slices.Clone(enclosure.Filled)
	slices.Sort(filled)
	assert.Equal(t, r.inside, filled, "the stream names every tile inside, once")

	start = time.Now()
	frame, err := proto.Marshal(planetmessage.TilesEnclosed(enclosure, version, false))
	rep.frameMarshal = time.Since(start)
	require.NoError(t, err)
	rep.frameBytes = len(frame)

	// The board: every tile inside is the attacker's but the ones a shield kept.
	struck := cpcolls.NewSet[uint32]()
	for _, s := range enclosure.Struck {
		struck.Add(s.Tile)
	}
	for _, tile := range r.inside {
		owner, _ := g.tiles.Owner(tile)
		if struck.Contains(tile) {
			assert.NotEqual(t, attacker, owner, "tile %d kept its flag with a shield", tile)
		} else if owner != attacker {
			assert.Equal(t, attacker, owner, "tile %d is taken", tile)
		}
	}

	// The ledger: one enclose event, then the relay's facts.
	require.NoError(t, g.ledger.Flush(t.Context()))
	sender := &countingSender{tiles: map[string]int{}}
	var cursor memoryCursor
	g.ledger.ReplaySaved(0, 1<<30, func(position ledger.Position, event ledger.Event) {
		if enclosing, ok := event.(ledger.Enclosing); ok && enclosing.Tile == r.closing {
			entry, err := enclosing.Entry()
			require.NoError(t, err)
			rep.ledgerBytes = len(entry.Payload)
			cursor.at = position
			sender.at = uint64(position)
		}
	})
	relay := relay_ledger_usecase.New(g.ledger, &cursor, g.world.borders, g.world.geography, sender)
	start = time.Now()
	for {
		told, err := relay.Execute(t.Context())
		require.NoError(t, err)
		if told < relay_ledger_usecase.Batch {
			break
		}
	}
	rep.relay = time.Since(start)
	rep.facts, rep.factBytes = sender.messages, sender.bytes
	assert.Equal(t, rep.filled-rep.struck, sender.tiles["taken"], "every tile taken is told, once")
	assert.Equal(t, rep.struck, sender.tiles["struck"], "every shield struck is told, once")
	assert.LessOrEqual(t, rep.facts, 2*(rep.filled/relay_ledger_usecase.PartTiles+1), "in parts, not one fact per tile")

	return rep
}

// Not a rule: a measure of the biggest pockets on the real map, run by hand (see docs/enclose.md).
// CP_GIANT=1 runs it; CP_GIANT_DUMP=<dir> also writes the rings the whole-stack run in e2e/ replays.
func TestGiantEnclosures(t *testing.T) {
	if os.Getenv("CP_GIANT") == "" && os.Getenv("CP_GIANT_DUMP") == "" {
		t.Skip("a measure, by hand: CP_GIANT=1")
	}
	w, err := loadWorld()
	require.NoError(t, err)

	if dir := os.Getenv("CP_GIANT_DUMP"); dir != "" {
		for name, r := range map[string]ring{
			"russia": w.ringAround(w.groundOf("ru"), w.firstTileOf("ru")),
			"africa": w.ringAround(w.groundOf(africa...), w.firstTileOf("cd")),
		} {
			encoded, err := json.Marshal(map[string]any{"wall": r.wall, "closing": r.closing, "inside": r.inside})
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(dir, name+".json"), encoded, 0o600)) //nolint:gosec // the dir the one running it names.
		}
	}

	var reports []report

	t.Run("russia, every tile held by ru, one in seven shielded", func(t *testing.T) {
		g := newGame(t, w)
		r := w.ringAround(w.groundOf("ru"), w.firstTileOf("ru"))
		g.paint(t, r.inside, "ru")
		for i, tile := range r.inside {
			if i%7 == 0 {
				_, err := g.tiles.Shield(t.Context(), tile, "ru", 3, clicks.PlacementID{})
				require.NoError(t, err)
			}
		}
		g.paint(t, r.wall, "ua")
		reports = append(reports, g.close(t, "russia (ru held, shields)", r, "ua", true))
	})

	t.Run("africa, blank", func(t *testing.T) {
		g := newGame(t, w)
		r := w.ringAround(w.groundOf(africa...), w.firstTileOf("cd"))
		g.paint(t, r.wall, "fr")
		reports = append(reports, g.close(t, "africa (blank)", r, "fr", true))
	})

	t.Run("africa, every country held by itself", func(t *testing.T) {
		g := newGame(t, w)
		r := w.ringAround(w.groundOf(africa...), w.firstTileOf("cd"))
		for _, tile := range r.inside {
			require.NoError(t, g.tiles.Set(player("00000000-0000-4000-8000-000000000001", "198.51.100.1"), tile, w.borders.CountryOf(tile)))
		}
		g.paint(t, r.wall, "fr")
		reports = append(reports, g.close(t, "africa (held by its flags)", r, "fr", true))
	})

	t.Run("afro-eurasia, every coast held", func(t *testing.T) {
		g := newGame(t, w)
		r := w.ringAround(func(uint32) bool { return true }, w.firstTileOf("ru"))
		g.paint(t, r.wall, "us")
		reports = append(reports, g.close(t, "afro-eurasia (blank)", r, "us", true))
	})

	t.Run("a ring with one gap: the search on an ordinary click", func(t *testing.T) {
		g := newGame(t, w)
		r := w.ringAround(w.groundOf("ru"), w.firstTileOf("ru"))
		// The gap: the wall tile furthest from the closing tile, on the coast, so the flood leaks out at the very end.
		gap := r.wall[len(r.wall)/2]
		var wall []uint32
		for _, tile := range r.wall {
			if tile != gap {
				wall = append(wall, tile)
			}
		}
		g.paint(t, wall, "ua")
		g.paint(t, []uint32{r.closing}, "ua")
		start := time.Now()
		pocket := g.terrain.PocketClosedBy(r.closing, "ua", g.rules.Enclose.MaxTiles)
		took := time.Since(start)
		t.Logf("leaky russia ring: search %s, took %d tiles (open: none)", took.Round(time.Microsecond), len(pocket.Inside()))
	})

	t.Run("ordinary clicks: the search's cost on every tile of the map", func(t *testing.T) {
		g := newGame(t, w)
		// Half the map painted in stripes of flags, as a busy battle leaves it.
		ctx := player("00000000-0000-4000-8000-000000000001", "198.51.100.1")
		for tile := uint32(1); tile <= mapTiles; tile++ {
			if tile%3 != 0 {
				require.NoError(t, g.tiles.Set(ctx, tile, w.borders.CountryOf(tile)))
			}
		}
		var costs []time.Duration
		for tile := uint32(1); tile <= mapTiles; tile += 13 {
			start := time.Now()
			g.terrain.PocketClosedBy(tile, "fr", g.rules.Enclose.MaxTiles)
			costs = append(costs, time.Since(start))
		}
		sort.Slice(costs, func(i, j int) bool { return costs[i] < costs[j] })
		t.Logf("search on %d tiles: p50 %s p99 %s p99.9 %s max %s", len(costs), costs[len(costs)/2], costs[len(costs)*99/100],
			costs[len(costs)*999/1000], costs[len(costs)-1])
	})

	for _, r := range reports {
		t.Log(r)
	}
}
