package stats_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats"
)

func columnsOf(header []string, record []string) map[string]string {
	cells := map[string]string{}
	for i, column := range header {
		cells[column] = record[i]
	}
	return cells
}

func TestTheTableHasAColumnForEachMetricAndEachPartOfTheMetadata(t *testing.T) {
	retention := &stats.Retention{HTTPStatus: 200, Body: json.RawMessage(`{"durationSeconds":14,"retentionCurve":[` +
		`{"elapsedVideoTimeRatio":0.5,"audienceWatchRatio":1.25,"relativeRetentionPerformance":0.6},` +
		`{"elapsedVideoTimeRatio":1,"audienceWatchRatio":0.5,"relativeRetentionPerformance":0.8}]}`)}
	rows := stats.RowsOf(aPost(), anAnswer(), map[string]*stats.Retention{"abc": retention}, takenAt)
	rows[0].Analytics = json.RawMessage(`{"platform":"tiktok","syncStatus":"synced","errorMessage":null,"analytics":{` +
		`"views":120,"completionRate":0.31,"impressionSources":{"forYou":0.84},"audienceCountries":{"US":0.6,"other":0.4},` +
		`"retentionCurve":{"0":1,"1":0.9}}}`)

	header, records := stats.Table(rows)

	require.Len(t, records, 2)
	assert.Equal(t, []string{
		"takenAt", "postId", "platform", "accountId", "platformPostId", "url", "publishedAt", "ageHours",
		"httpStatus", "postStatus", "syncStatus", "errorMessage",
	}, header[:12], "the columns of every row come first")
	tiktok := columnsOf(header, records[0])
	assert.Equal(t, "2026-10-24T05:41:00Z", tiktok["takenAt"])
	assert.Equal(t, "37.02", tiktok["ageHours"])
	assert.Equal(t, "202", tiktok["httpStatus"])
	assert.Equal(t, "synced", tiktok["syncStatus"], "the platform's own state wins over the post's")
	assert.Equal(t, "120", tiktok["metrics.views"])
	assert.Equal(t, "0.31", tiktok["metrics.completionRate"], "a number as zernio wrote it")
	assert.Equal(t, "0.84", tiktok["metrics.impressionSources.forYou"])
	assert.JSONEq(t, `{"US":0.6,"other":0.4}`, tiktok["metrics.audienceCountries"])
	assert.JSONEq(t, `{"0":1,"1":0.9}`, tiktok["metrics.retentionCurve"])
	assert.Equal(t, "fr", tiktok["meta.storyFlag"])
	assert.JSONEq(t, `["fr","pw"]`, tiktok["meta.storyCountries"])
	assert.Equal(t, "short", tiktok["meta.record.clip.variants.length"])
	assert.Empty(t, tiktok["youtube.retentionStatus"])

	youtube := columnsOf(header, records[1])
	assert.Equal(t, "pending", youtube["syncStatus"])
	assert.Empty(t, youtube["metrics.views"], "a column one row has is empty on the others")
	assert.Equal(t, "200", youtube["youtube.retentionStatus"])
	assert.Equal(t, "14", youtube["youtube.durationSeconds"])
	assert.Equal(t, "2", youtube["youtube.points"])
	assert.Equal(t, "0.7000", youtube["youtube.meanRelativeRetentionPerformance"])
	assert.Contains(t, youtube["youtube.retentionCurve"], `"audienceWatchRatio":1.25`)
}

func TestNoRowIsATableOfItsHeaderAlone(t *testing.T) {
	header, records := stats.Table(nil)

	assert.Len(t, header, 12)
	assert.Empty(t, records)
}
