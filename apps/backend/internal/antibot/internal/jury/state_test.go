package jury

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/shadowban"
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

func newJury(clock cptime.Clock, onFlag func(detect.Report), watchdog detect.Watchdog) *Jury {
	banner := shadowban.New(shadowban.Config{Enforce: true}, clock, nil)
	return New(Config{TrackWindow: time.Hour}, banner, clock, onFlag, watchdog)
}

func TestTheCallerAndItsOpinionsSurviveASaveAndLoad(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	watchdog := &saying{
		verdict: detect.Suspect,
		evidence: detect.Evidence{Rule: "stride", Fields: []detect.Field{
			{Key: "share", Value: 0.8},
			{Key: "spread", Value: 120 * time.Millisecond},
			{Key: "tiles", Value: []uint32{1, 2}},
		}},
	}
	j := newJury(clock, nil, watchdog)

	for tile := range uint32(20) {
		clock.Advance(time.Second)
		j.Inspect(detect.Click{Scope: "caller", Tile: tile, Country: "FR", At: clock.Now()})
	}

	savedAt := clock.Now()
	data, err := j.Save()
	require.NoError(t, err)

	clock.Advance(40 * time.Second)

	var reports []detect.Report
	restarted := newJury(clock, func(report detect.Report) { reports = append(reports, report) }, watchdog)
	require.NoError(t, restarted.Load(data))
	restarted.Resume(detect.Outage{From: savedAt, To: clock.Now()})

	loaded := restarted.callers["caller"].opinions["saying"]
	assert.Equal(t, j.callers["caller"].opinions["saying"].String(), loaded.String(), "the log line reads the same")
	assert.Equal(t, detect.Suspect, loaded.Verdict)

	clock.Advance(time.Second)
	watchdog.verdict = detect.Certain
	require.True(t, restarted.Inspect(detect.Click{Scope: "caller", Tile: 20, Country: "FR", At: clock.Now()}))
	require.Len(t, reports, 1)

	report := reports[0]
	assert.Equal(t, 21, report.Clicks)
	assert.Equal(t, 21, report.TopCountryClicks)
	assert.Equal(t, 20*time.Second, report.ActiveFor, "the outage is not time the caller was seen")
	assert.Equal(t, time.Second, report.LongestGap, "a restart is not the caller stopping")
	assert.Len(t, report.Tiles, keptTiles)
}

func TestForgetDropsSilentCallersAndOldOpinions(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	j := newJury(clock, nil, &saying{verdict: detect.Clear})
	j.Inspect(detect.Click{Scope: "silent", Tile: 1, Country: "FR", At: clock.Now()})
	clock.Advance(time.Hour)
	j.Inspect(detect.Click{Scope: "active", Tile: 1, Country: "FR", At: clock.Now()})

	j.Forget(clock.Now().Add(-time.Minute))

	require.Len(t, j.callers, 1)
	assert.Contains(t, j.callers, "active")
}
