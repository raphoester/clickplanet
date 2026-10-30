package stats_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats"
)

func curveAt(ageHours float64) *stats.Retention {
	return &stats.Retention{HTTPStatus: 200, Body: json.RawMessage(`{"retentionCurve":[{"audienceWatchRatio":1}]}`), AgeHours: ageHours}
}

func TestACurveIsDueOncePastTwoDaysAndOncePastAWeekUntilOneComes(t *testing.T) {
	const hour = time.Hour
	empty := &stats.Retention{HTTPStatus: 200, Body: json.RawMessage(`{"retentionCurve":[]}`), AgeHours: 49}

	for _, example := range []struct {
		name string
		age  time.Duration
		held *stats.Retention
		due  bool
	}{
		{"too young for YouTube to have measured it", 47 * hour, nil, false},
		{"past 2 days, none held", 49 * hour, nil, true},
		{"past 2 days, an empty one held", 73 * hour, empty, true},
		{"past 2 days, one read past 2 days", 100 * hour, curveAt(49), false},
		{"past a week, one read before it", 169 * hour, curveAt(49), true},
		{"past a week, one read past it", 400 * hour, curveAt(170), false},
		{"past a week, none held", 400 * hour, nil, true},
	} {
		assert.Equal(t, example.due, stats.RetentionDue(example.age, example.held), example.name)
	}
}

func TestANewReadWithNoCurveKeepsTheOneHeldAndSaysWhy(t *testing.T) {
	held := curveAt(49)
	refused := &stats.Retention{HTTPStatus: 412, Body: json.RawMessage(`{"code":"youtube_analytics_scope_missing"}`)}
	fresh := curveAt(170)

	assert.Equal(t, stats.RetentionState{Retention: held, Missing: "412 youtube_analytics_scope_missing"}, stats.RetentionAfter(held, refused))
	assert.Equal(t, stats.RetentionState{Retention: fresh}, stats.RetentionAfter(held, fresh))
	assert.Equal(t, stats.RetentionState{Retention: refused}, stats.RetentionAfter(nil, refused), "with none held, the refusal is the answer")
}

func TestARetentionReadSaysWhenAndAtWhatAge(t *testing.T) {
	read := stats.Retention{HTTPStatus: 200}.ReadOn(stats.Target{PublishedAt: publishedAt}, takenAt)

	assert.Equal(t, takenAt, read.ReadAt)
	assert.InDelta(t, 37.02, read.AgeHours, 0.001)
}

func TestTheCurvesHeldAreTheLastSnapshotsAndAnOldRowWasReadWithItsRow(t *testing.T) {
	readBefore := takenAt.Add(-48 * time.Hour)
	rows := []stats.Row{
		{TakenAt: takenAt, AgeHours: 60, PlatformPostID: "old", Retention: &stats.Retention{HTTPStatus: 200}},
		{TakenAt: takenAt, AgeHours: 80, PlatformPostID: "carried", Retention: &stats.Retention{ReadAt: readBefore, AgeHours: 32}},
		{TakenAt: takenAt, PlatformPostID: "none"},
		{TakenAt: takenAt, Platform: "tiktok"},
	}

	held := stats.HeldRetentions(rows)

	require.Len(t, held, 2)
	assert.Equal(t, takenAt, held["old"].ReadAt)
	assert.InDelta(t, 60, held["old"].AgeHours, 0.001)
	assert.Equal(t, readBefore, held["carried"].ReadAt)
	assert.True(t, rows[0].Retention.ReadAt.IsZero(), "the rows read are not changed")
}

func TestTheReportCountsTheCurvesRefusedNowAndThoseMissing(t *testing.T) {
	refused := &stats.Retention{HTTPStatus: 200, Body: json.RawMessage(`{"retentionCurve":[]}`), ReadAt: takenAt}
	rows := []stats.Row{
		{TakenAt: takenAt, Platform: "youtube", Retention: refused},
		{TakenAt: takenAt, Platform: "youtube", Retention: &stats.Retention{HTTPStatus: 200, ReadAt: takenAt.Add(-time.Hour)}},
		{TakenAt: takenAt, Platform: "youtube", Retention: curveAt(50), RetentionMissing: stats.QuotaSpent},
		{TakenAt: takenAt, Platform: "youtube", RetentionMissing: stats.QuotaSpent},
	}

	report := stats.ReportOf(stats.Accounts{}, rows)

	assert.Equal(t, 1, report.Curves, "a curve carried is a curve")
	assert.Equal(t, map[string]int{"empty": 1}, report.CurvesRefused, "only what was read now")
	assert.Equal(t, map[string]int{stats.QuotaSpent: 2}, report.RetentionMissing)
}
