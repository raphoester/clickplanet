package jury

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type saying struct {
	verdict  detect.Verdict
	evidence detect.Evidence
}

func (s *saying) Name() string                                         { return "saying" }
func (s *saying) Attempted(detect.Click)                               {}
func (s *saying) Committed(detect.Click)                               {}
func (s *saying) Watch(detect.Click) (detect.Verdict, detect.Evidence) { return s.verdict, s.evidence }

func newJury(clock cptime.Clock, hooks Hooks, watchdog detect.Watchdog) *Jury {
	banner := bans.NewBook(bans.Config{Enforce: true}, 72*time.Hour, clock, bans.NewMemoryStore(), bans.NewMemoryStore(), cpatomic.FakeTransactioner{}, &bans.MemoryEnforcer{})
	return New(Config{TrackWindow: time.Hour}, banner, clock, hooks, watchdog)
}

func TestTheCallerAndItsOpinionsSurviveASaveAndLoad(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	watchdog := &saying{
		verdict: detect.Suspect,
		evidence: detect.Evidence{Rule: "stride", Zeal: true, Fields: []detect.Field{
			{Key: "share", Value: 0.8},
			{Key: "spread", Value: 120 * time.Millisecond},
			{Key: "tiles", Value: []uint32{1, 2}},
		}},
	}
	j := newJury(clock, Hooks{}, watchdog)

	for tile := range uint32(20) {
		clock.Advance(time.Second)
		inspect(t, j, detect.Click{Scope: "caller", Tile: tile, Country: "FR", At: clock.Now()})
	}

	savedAt := clock.Now()
	data, err := j.Save()
	require.NoError(t, err)

	clock.Advance(40 * time.Second)

	var (
		reports []detect.Report
		rises   []detect.Verdict
	)
	restarted := newJury(clock, Hooks{
		OnFlag: func(report detect.Report) { reports = append(reports, report) },
		OnRise: func(_ string, level detect.Verdict) { rises = append(rises, level) },
	}, watchdog)
	require.NoError(t, restarted.Load(data))
	restarted.Resume(detect.Outage{From: savedAt, To: clock.Now()})

	loaded := restarted.callers["caller"].opinions["saying"]
	assert.Equal(t, j.callers["caller"].opinions["saying"].String(), loaded.String(), "the log line reads the same")
	assert.Equal(t, detect.Suspect, loaded.Verdict)
	assert.True(t, loaded.Evidence.Zeal, "zeal still counts once after a restart")

	clock.Advance(time.Second)
	watchdog.verdict = detect.Certain
	inspect(t, restarted, detect.Click{Scope: "caller", Tile: 20, Country: "FR", At: clock.Now()})
	require.Len(t, reports, 1)
	assert.Equal(t, []detect.Verdict{detect.Certain}, rises, "suspect was already standing before the restart")

	report := reports[0]
	assert.Equal(t, 21, report.Clicks)
	assert.Equal(t, 21, report.TopCountryClicks)
	assert.Equal(t, 20*time.Second, report.ActiveFor, "the outage is not time the caller was seen")
	assert.Equal(t, time.Second, report.LongestGap, "a restart is not the caller stopping")
	assert.Len(t, report.Tiles, keptTiles)
}

func TestAReadingOfTurnsStillCountsForNothingAfterARestart(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 10, 19, 18, 0, 0, 0, time.UTC))

	watchdog := &saying{verdict: detect.Suspect, evidence: detect.Evidence{Rule: "rotation", Turns: true}}
	j := newJury(clock, Hooks{}, watchdog)
	inspect(t, j, detect.Click{Scope: "caller", Tile: 1, Country: "FR", At: clock.Now()})

	data, err := j.Save()
	require.NoError(t, err)

	restarted := newJury(clock, Hooks{}, watchdog)
	require.NoError(t, restarted.Load(data))

	assert.True(t, restarted.callers["caller"].opinions["saying"].Evidence.Turns)
	assert.Zero(t, restarted.Examine("caller", "").Suspects)
}

func TestForgetDropsSilentCallersAndOldOpinions(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	j := newJury(clock, Hooks{}, &saying{verdict: detect.Clear})
	inspect(t, j, detect.Click{Scope: "silent", Tile: 1, Country: "FR", At: clock.Now()})
	clock.Advance(time.Hour)
	inspect(t, j, detect.Click{Scope: "active", Tile: 1, Country: "FR", At: clock.Now()})

	j.Forget(clock.Now().Add(-time.Minute))

	require.Len(t, j.callers, 1)
	assert.Contains(t, j.callers, "active")
}

func TestWhereEachAccountClickedLastSurvivesASaveAndLoad(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC))
	j := newJury(clock, Hooks{}, &saying{verdict: detect.Clear})
	inspect(t, j, detect.Click{Scope: "old", Account: "gone", Tile: 1, Country: "FR", At: clock.Now()})
	clock.Advance(time.Hour)
	inspect(t, j, detect.Click{Scope: "home", Account: "player", Tile: 1, Country: "FR", At: clock.Now()})

	data, err := j.Accounts().Save()
	require.NoError(t, err)

	restarted := newJury(clock, Hooks{}, &saying{verdict: detect.Clear})
	require.NoError(t, restarted.Accounts().Load(data))
	restarted.Accounts().Forget(clock.Now().Add(-time.Minute))

	assert.Equal(t, "home", restarted.Examine("", "player").Scope)
	assert.Empty(t, restarted.Examine("", "gone").Scope, "a sighting past the retention is forgotten")
}
