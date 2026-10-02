package scraper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/scraper"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// chunk is one GetMap of ten thousand tiles, as the bot of 2026-09-15 read them.
const chunk = 10000.0 / 262119

type harness struct {
	watchdog *scraper.Watchdog
	clock    *cptime.FixedClock
}

func newHarness() *harness {
	h := &harness{clock: cptime.NewFixedClock(time.Date(2026, 9, 15, 21, 11, 0, 0, time.UTC))}
	h.watchdog = scraper.New(scraper.Config{
		MinMaps:       5,
		CertainMaps:   15,
		CertainOffMap: 2,
		TrackWindow:   15 * time.Minute,
	}, h.clock, func(float64) {})
	return h
}

// pageLoad is the web app opening: the stream, then the whole map in 26 chunks back to back.
func (h *harness) pageLoad(scope string) {
	h.watchdog.Listened(scope)
	for range 26 {
		h.clock.Advance(80 * time.Millisecond)
		h.watchdog.Fetched(scope, 1.0/26, false)
	}
}

// walk is the same volume read off the map's own lattice: one chunk of it starts at tile 0.
func (h *harness) walk(scope string) {
	for i := range 26 {
		h.clock.Advance(80 * time.Millisecond)
		h.watchdog.Fetched(scope, 1.0/26, i == 0)
	}
}

func (h *harness) click(scope string) (detect.Verdict, detect.Evidence) {
	click := detect.Click{Scope: scope, Tile: 1, Country: "dz", At: h.clock.Now()}
	h.watchdog.Attempted(click)
	return h.watchdog.Watch(click)
}

// poll clicks every 1.1s and reads a chunk after each click, and says when each level was first read.
func (h *harness) poll(scope string, d time.Duration) (detect.Verdict, map[detect.Verdict]time.Duration) {
	start := h.clock.Now()
	first := map[detect.Verdict]time.Duration{}

	var verdict detect.Verdict
	for h.clock.Now().Sub(start) < d {
		h.clock.Advance(1100 * time.Millisecond)
		verdict, _ = h.click(scope)
		if _, seen := first[verdict]; !seen {
			first[verdict] = h.clock.Now().Sub(start)
		}
		h.watchdog.Fetched(scope, chunk, false)
	}

	return verdict, first
}

func TestTheMapScraperIsCaught(t *testing.T) {
	h := newHarness()

	h.pageLoad("bot")
	verdict, first := h.poll("bot", 10*time.Minute)

	require.Equal(t, detect.Certain, verdict)
	assert.Less(t, first[detect.Suspect], 3*time.Minute, "five maps no stream explains")
	assert.Less(t, first[detect.Certain], 8*time.Minute, "fifteen of them, a map every thirty seconds")

	_, evidence := h.click("bot")
	assert.Equal(t, "poll", evidence.Rule)
	assert.Contains(t, evidence.Fields, detect.Field{Key: "streams", Value: 1})
	assert.Contains(t, evidence.Fields, detect.Field{Key: "offMap", Value: 0}, "this one stayed on the lattice")
}

func TestAPageLoadIsClear(t *testing.T) {
	h := newHarness()

	h.pageLoad("player")

	for range 800 {
		h.clock.Advance(1100 * time.Millisecond)
		verdict, _ := h.click("player")
		require.Equal(t, detect.Clear, verdict, "the map was read once and followed after that")
	}
}

func TestReloadingOverAndOverIsClear(t *testing.T) {
	h := newHarness()

	for range 30 {
		h.pageLoad("reloader")
		h.clock.Advance(30 * time.Second)

		verdict, _ := h.click("reloader")
		require.Equal(t, detect.Clear, verdict, "each reload opens a stream too; so does a script that copies it")
	}
}

func TestARaidBehindOneAddressIsClear(t *testing.T) {
	h := newHarness()

	for range 60 {
		h.pageLoad("carrier nat")
		h.clock.Advance(5 * time.Second)
	}

	verdict, _ := h.click("carrier nat")
	assert.Equal(t, detect.Clear, verdict, "sixty players, sixty page loads")
}

func TestReadsOlderThanTheWindowStopCounting(t *testing.T) {
	h := newHarness()

	verdict, _ := h.poll("bot", 10*time.Minute)
	require.Equal(t, detect.Certain, verdict)

	h.clock.Advance(16 * time.Minute)

	verdict, _ = h.click("bot")
	assert.Equal(t, detect.Clear, verdict)
}

func TestStreamsDoNotBankCreditPastTheWindow(t *testing.T) {
	h := newHarness()

	for range 50 {
		h.watchdog.Listened("bot")
	}
	h.clock.Advance(16 * time.Minute)

	verdict, _ := h.poll("bot", 10*time.Minute)
	assert.Equal(t, detect.Certain, verdict)
}

func TestAnotherCallersReadsAreNotTheirs(t *testing.T) {
	h := newHarness()

	verdict, _ := h.poll("bot", 10*time.Minute)
	require.Equal(t, detect.Certain, verdict)

	verdict, _ = h.click("someone else")
	assert.Equal(t, detect.Clear, verdict)
}

func TestAStreamBeforeEveryReadBuysNothingOffTheLattice(t *testing.T) {
	h := newHarness()

	for range 7 {
		h.watchdog.Listened("bot")
		h.walk("bot")
		h.clock.Advance(30 * time.Second)
	}

	verdict, evidence := h.click("bot")
	assert.Equal(t, detect.Certain, verdict, "a stream per read is no page load when the read is off the map")
	assert.Equal(t, "offMap", evidence.Rule)
	assert.Contains(t, evidence.Fields, detect.Field{Key: "offMap", Value: 7})
}

func TestOneReadOffTheMapIsNotEnoughOnItsOwn(t *testing.T) {
	h := newHarness()

	h.watchdog.Fetched("curious", chunk, true)
	h.pageLoad("curious")

	verdict, _ := h.click("curious")
	assert.Equal(t, detect.Clear, verdict, "losing the credit still leaves a page load under the bound")
}

// cachedWalk is the bot of 2026-09-16: it reads the map from tile 0 and past the end, then
// keeps the copy and clicks off it, so the reads stop long before the ban would have to fire.
func (h *harness) cachedWalk(scope string) {
	for i := range 26 {
		h.clock.Advance(80 * time.Millisecond)
		h.watchdog.Fetched(scope, 1.0/26, i == 0 || i == 25)
	}
}

func TestTheWalkOffTheLatticeIsCaughtBeforeItPaints(t *testing.T) {
	h := newHarness()

	h.watchdog.Listened("bot")
	h.cachedWalk("bot")

	verdict, evidence := h.click("bot")
	require.Equal(t, detect.Certain, verdict, "two maps is far under minMaps, and the walk is still proof")
	assert.Equal(t, "offMap", evidence.Rule)
}

func TestACallerThatStopsReadingKeepsItsVerdictForTheWindow(t *testing.T) {
	h := newHarness()

	h.watchdog.Listened("bot")
	h.cachedWalk("bot")

	for range 10 {
		h.clock.Advance(80 * time.Second)
		verdict, _ := h.click("bot")
		require.Equal(t, detect.Certain, verdict, "it reads nothing more; the reads it made still stand")
	}
}

func TestTheOffMapReadsMustBeInsideTheWindow(t *testing.T) {
	h := newHarness()

	h.watchdog.Fetched("curious", chunk, true)
	h.clock.Advance(16 * time.Minute)
	h.watchdog.Fetched("curious", chunk, true)

	verdict, _ := h.click("curious")
	assert.Equal(t, detect.Clear, verdict, "one stray read a quarter of an hour apart is no walk")
}
