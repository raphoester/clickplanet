package recent_history_query_test

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/embedded_board"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/planetmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

// A day as busy as production on 2026-11-11: ~111,000 changes of owner by ~98 accounts. The clicks do not grow with the
// board, so neither do the takes: a front of the same ground holds more tiles.
const (
	flags         = 200
	accounts      = 98
	takesPerStep  = 2300
	strikePerStep = 600
	answerBudget  = 1 << 20
)

type day struct {
	geography *clicks.Geography
	random    *rand.Rand
	tiles     int
	owners    []string
	hotSpots  [][]uint32
	events    []ledger.Event
	spot      int
}

func TestTheAnswerOfADayAsBusyAsProductionsFitsItsBudget(t *testing.T) {
	for _, board := range []clicks.BoardName{"earth-1", "earth-2"} {
		t.Run(string(board), func(t *testing.T) { aDayAsBusyAsProductionsFits(t, board) })
	}
}

func aDayAsBusyAsProductionsFits(t *testing.T, board clicks.BoardName) {
	ground, err := embedded_board.Shipped(board)
	require.NoError(t, err)
	geography := ground.Geography()
	mapTiles := int(ground.Tiles())
	density := float64(mapTiles) / 260503
	random := rand.New(rand.NewPCG(11, 11)) //nolint:gosec // a fixed draw, not a secret.
	busy := &day{
		geography: geography, random: random, tiles: mapTiles, owners: make([]string, mapTiles+1), spot: int(150 * density),
	}
	busy.hold()
	opening := slices.Clone(busy.owners)
	busy.fight()

	rewound, err := ledger.NewRewind(since, until, 30*time.Minute, clicks.WorldStage)
	require.NoError(t, err)
	for _, event := range busy.events {
		entry, err := event.Entry()
		require.NoError(t, err)
		entry.Stage = clicks.WorldStage
		require.NoError(t, rewound.Record(entry))
	}
	history := planetmessage.MapHistory(rewound.Frames(denseOf(busy.owners)), ledger.InEra(0, rewound.Hits()), nil)
	frames := history.GetFrames()

	whole := frames[0].GetWhole()
	replayed := make([]string, mapTiles+1)
	for i := 0; i+1 < len(whole.GetTiles()); i += 2 {
		replayed[whole.GetStartTileId()+uint32(i/2)] = whole.GetCodes()[binary.LittleEndian.Uint16(whole.GetTiles()[i:])] //nolint:gosec // within the map.
	}
	assert.True(t, slices.Equal(opening, replayed), "the first frame, whole, is the map the day opened on")
	changes := 0
	for _, frame := range frames[1:] {
		for _, owner := range frame.GetOwners() {
			replayed[owner.GetTile()] = owner.GetCountryId()
			changes++
		}
	}
	assert.True(t, slices.Equal(busy.owners, replayed), "every frame played after it is the map now")

	answer, err := proto.Marshal(&planetv1.GetRecentMapHistoryResponse{History: history})
	require.NoError(t, err)
	var zipped bytes.Buffer
	writer := gzip.NewWriter(&zipped)
	_, err = writer.Write(answer)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	t.Logf("%d tiles held at the start, %d changes in %d frames, %d tiles hit: %d KB, %d KB gzipped",
		len(opening)-1-countOf(opening[1:], ""), changes, len(frames), len(rewound.Hits()), len(answer)/1024, zipped.Len()/1024)
	assert.LessOrEqual(t, zipped.Len(), answerBudget)
}

func (d *day) hold() {
	queue := make([]uint32, 0, d.tiles)
	for i := range flags {
		seed := uint32(d.random.IntN(d.tiles)) + 1 //nolint:gosec // within the map.
		if d.owners[seed] == "" {
			d.owners[seed] = flagOf(i)
			queue = append(queue, seed)
		}
	}
	for len(queue) > 0 {
		tile := queue[0]
		queue = queue[1:]
		for _, next := range d.geography.Neighbours(tile) {
			if d.owners[next] == "" {
				d.owners[next] = d.owners[tile]
				queue = append(queue, next)
			}
		}
	}
	for range d.tiles * 3 / 100 {
		d.owners[d.random.IntN(d.tiles)+1] = ""
	}
	for range 30 {
		d.hotSpots = append(d.hotSpots, d.ball(uint32(d.random.IntN(d.tiles))+1, d.spot)) //nolint:gosec // within the map.
	}
}

type timed struct {
	at    time.Time
	event ledger.Event
}

func (d *day) fight() {
	for step := range 48 {
		start := since.Add(time.Duration(step) * 30 * time.Minute)
		var events []timed
		takes := 0
		for takes < takesPerStep {
			account := int(float64(accounts) * d.random.Float64() * d.random.Float64())
			flag := flagOf(account % 40)
			spot := d.hotSpots[d.random.IntN(len(d.hotSpots))]
			for _, tile := range d.patch(spot, flag, 5+d.random.IntN(55)) {
				at := start.Add(time.Duration(min(takes, takesPerStep)*1790/takesPerStep) * time.Second)
				events = append(events, timed{at: at, event: ledger.Taking{
					Tile: tile, Scope: "203.0.113.7", Account: accountOf(account), Country: flag, Previous: d.owners[tile], At: at,
				}})
				d.owners[tile] = flag
				takes++
			}
		}
		for range strikePerStep {
			spot := d.hotSpots[d.random.IntN(len(d.hotSpots))]
			tile := spot[d.random.IntN(len(spot))]
			at := start.Add(time.Duration(d.random.IntN(1799)) * time.Second)
			events = append(events, timed{at: at, event: ledger.Striking{
				Tile: tile, Scope: "203.0.113.7", Account: accountOf(0), Country: "zz", Owner: d.owners[tile], At: at,
			}})
		}
		slices.SortStableFunc(events, func(a, b timed) int { return a.at.Compare(b.at) })
		for _, event := range events {
			d.events = append(d.events, event.event)
		}
	}
}

func (d *day) ball(centre uint32, size int) []uint32 {
	seen := cpcolls.NewSet(centre)
	ball := []uint32{centre}
	for i := 0; i < len(ball) && len(ball) < size; i++ {
		for _, next := range d.geography.Neighbours(ball[i]) {
			if !seen.Contains(next) {
				seen.Add(next)
				ball = append(ball, next)
			}
		}
	}
	return ball
}

func (d *day) patch(spot []uint32, flag string, size int) []uint32 {
	within := cpcolls.NewSet(spot...)
	start := spot[d.random.IntN(len(spot))]
	seen := cpcolls.NewSet(start)
	queue, patch := []uint32{start}, []uint32{}
	for len(queue) > 0 && len(patch) < size {
		tile := queue[0]
		queue = queue[1:]
		if d.owners[tile] != flag {
			patch = append(patch, tile)
		}
		for _, next := range d.geography.Neighbours(tile) {
			if within.Contains(next) && !seen.Contains(next) {
				seen.Add(next)
				queue = append(queue, next)
			}
		}
	}
	return patch
}

func flagOf(i int) string {
	return fmt.Sprintf("%c%c", 'a'+i/26, 'a'+i%26)
}

func accountOf(i int) string {
	return fmt.Sprintf("0b6d4f7e-5d7c-4a36-9a51-%012d", i)
}

func denseOf(owners []string) clicks.DenseBatch {
	batch := clicks.DenseBatch{Start: 1, Codes: []string{""}}
	codes := map[string]uint16{"": 0}
	for _, owner := range owners[1:] {
		code, known := codes[owner]
		if !known {
			code = uint16(len(batch.Codes))
			codes[owner] = code
			batch.Codes = append(batch.Codes, owner)
		}
		batch.Tiles = append(batch.Tiles, byte(code), byte(code>>8))
	}
	return batch
}

func countOf(owners []string, owner string) int {
	count := 0
	for _, held := range owners {
		if held == owner {
			count++
		}
	}
	return count
}
