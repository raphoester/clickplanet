package metronome_test

import (
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/metronome"
)

const network = "2001:db8:10:9::/64"

type pace func(r *rand.Rand, last time.Time, try int) time.Time

func gapsOf(gap func(*rand.Rand, int) time.Duration) pace {
	return func(r *rand.Rand, last time.Time, try int) time.Time { return last.Add(gap(r, try)) }
}

func byHand(r *rand.Rand, _ int) time.Duration {
	return 300*time.Millisecond + time.Duration(r.Int64N(int64(600*time.Millisecond)))
}

func byMachine(r *rand.Rand, _ int) time.Duration {
	return 890*time.Millisecond + time.Duration(r.Int64N(int64(120*time.Millisecond)))
}

func onTheSecond(r *rand.Rand, last time.Time, try int) time.Time {
	tick := last.Truncate(time.Second).Add(time.Second)
	if try > 0 && try%60 == 0 {
		tick = tick.Add(time.Duration(330+r.IntN(90)) * time.Second)
	}
	return tick.Add(135*time.Millisecond + time.Duration(r.IntN(10))*time.Millisecond)
}

type reading struct {
	try      int
	since    time.Duration
	verdict  detect.Verdict
	evidence detect.Evidence
}

func (r reading) duration(t *testing.T, key string) time.Duration {
	t.Helper()

	for _, field := range r.evidence.Fields {
		if field.Key == key {
			value, ok := field.Value.(time.Duration)
			require.True(t, ok, key)
			return value
		}
	}

	require.Fail(t, "the evidence has no "+key)
	return 0
}

type browser struct {
	account string
	pace    pace
	random  *rand.Rand

	next  time.Time
	tries int

	first map[detect.Verdict]reading
	last  reading
}

type household struct {
	watchdog *metronome.Watchdog
	start    time.Time
	now      time.Time
	browsers []*browser
}

func newHousehold(rules metronome.Rules, config metronome.Config) *household {
	h := newHarness(rules, config)
	return &household{watchdog: h.watchdog, start: h.clock.Now(), now: h.clock.Now()}
}

func (h *household) browser(account string, pace pace) *browser {
	b := &browser{
		account: account,
		pace:    pace,
		random:  rand.New(rand.NewPCG(uint64(len(h.browsers)), 9)), //nolint:gosec // a test draw, not a secret.
		first:   map[detect.Verdict]reading{},
	}
	b.next = pace(b.random, h.start, 0)

	h.browsers = append(h.browsers, b)
	return b
}

func (h *household) play(lasting time.Duration) {
	h.playUntil(func() bool { return h.now.Sub(h.start) >= lasting })
}

func (h *household) playUntil(done func() bool) {
	for !done() {
		b := slices.MinFunc(h.browsers, func(a, b *browser) int { return a.next.Compare(b.next) })
		h.now = b.next

		click := detect.Click{Scope: network, Account: b.account, At: h.now}
		h.watchdog.Attempted(click)
		verdict, evidence := h.watchdog.Watch(click)

		b.tries++
		b.last = reading{try: b.tries, since: h.now.Sub(h.start), verdict: verdict, evidence: evidence}
		if _, read := b.first[verdict]; !read {
			b.first[verdict] = b.last
		}

		b.next = b.pace(b.random, h.now, b.tries)
	}
}

func productionCadence() (metronome.Rules, metronome.Config) {
	rules := shipped()
	rules.MaxGap, rules.MaxSpread = 7*time.Second, 400*time.Millisecond
	return rules, metronome.Config{TrackWindow: 15 * time.Minute}
}

func TestHandsBehindOneNetworkAreEachJudgedAlone(t *testing.T) {
	h := newHousehold(productionCadence())
	for _, account := range []string{"one", "two", "three"} {
		h.browser(account, gapsOf(byHand))
	}

	h.play(3 * time.Minute)

	for _, b := range h.browsers {
		require.Greater(t, b.tries, 240, "twice the clicks a run needs to be read")
		assert.NotContains(t, b.first, detect.Suspect, "%s clicks by hand, whoever clicks beside it", b.account)
	}
}

func TestWithNoAccountANetworkIsStillOneCaller(t *testing.T) {
	h := newHousehold(productionCadence())
	for range 3 {
		h.browser("", gapsOf(byHand))
	}

	h.play(3 * time.Minute)

	suspect, read := h.browsers[0].first[detect.Suspect]
	require.True(t, read, "nothing tells three hands on one stream apart")
	assert.Equal(t, "cadence", suspect.evidence.Rule)
	assert.Less(t, suspect.duration(t, "median"), 300*time.Millisecond, "shorter than any gap one of them made")
	assert.LessOrEqual(t, suspect.duration(t, "spread"), 400*time.Millisecond, "and a short gap has a small spread")
}

func TestALoopBehindABusyNetworkIsSuspectThenCertain(t *testing.T) {
	h := newHousehold(productionCadence())
	loop := h.browser("loop", gapsOf(byMachine))
	neighbours := []*browser{h.browser("one", gapsOf(hand)), h.browser("two", gapsOf(hand))}

	h.play(35 * time.Minute)

	suspect, read := loop.first[detect.Suspect]
	require.True(t, read)
	assert.Equal(t, 120, suspect.try, "as soon as its own run can be read")
	assert.Equal(t, "cadence", suspect.evidence.Rule)
	assert.InDelta(t, 950*time.Millisecond, suspect.duration(t, "median"), float64(30*time.Millisecond), "its own gaps, not the network's")

	certain, read := loop.first[detect.Certain]
	require.True(t, read)
	assert.GreaterOrEqual(t, certain.since, 30*time.Minute)
	assert.Equal(t, detect.Certain, loop.last.verdict)

	for _, b := range neighbours {
		require.Greater(t, b.tries, 900, "as many clicks as the loop needed")
		assert.NotContains(t, b.first, detect.Suspect, "%s is not judged on the loop beside it", b.account)
		assert.NotContains(t, b.first, detect.Certain)
	}
}

func TestARandomSleepBehindABusyNetworkIsStillRead(t *testing.T) {
	h := newHousehold(shaped())
	loop := h.browser("loop", gapsOf(randomSleep))
	neighbours := []*browser{h.browser("one", gapsOf(hand)), h.browser("two", gapsOf(hand))}

	h.playUntil(func() bool { return loop.tries >= 700 })

	require.Equal(t, detect.Suspect, loop.last.verdict, "the hands beside it no longer bend the shape of its gaps")
	assert.Equal(t, "shape", loop.last.evidence.Rule)

	for _, b := range neighbours {
		require.Greater(t, b.tries, 500, "enough gaps for a full window")
		assert.NotContains(t, b.first, detect.Suspect)
	}
}

func TestAHiddenTabBehindABusyNetworkKeepsItsBeat(t *testing.T) {
	h := newHousehold(clocked())
	tab := h.browser("tab", onTheSecond)
	neighbours := []*browser{h.browser("one", gapsOf(hand)), h.browser("two", gapsOf(hand))}

	h.playUntil(func() bool { return tab.tries >= 130 })

	require.Equal(t, detect.Suspect, tab.last.verdict, "the hands beside it no longer blur where in the second it lands")
	assert.Equal(t, "clock", tab.last.evidence.Rule)

	for _, b := range neighbours {
		require.Greater(t, b.tries, 120, "enough tries for a full window")
		assert.NotContains(t, b.first, detect.Suspect)
	}
}
