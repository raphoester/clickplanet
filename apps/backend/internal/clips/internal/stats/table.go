package stats

import (
	"bytes"
	"encoding/json"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

// The columns every table starts with, in this order. The others are what the rows hold, by name.
var fixedColumns = []string{
	"takenAt", "postId", "platform", "accountId", "platformPostId", "url", "publishedAt", "ageHours",
	"httpStatus", "postStatus", "syncStatus", "errorMessage",
}

// Kept as one cell of JSON: too many keys to be columns, and different keys on every row.
var wideKeys = []string{"retentionCurve", "audienceCountries"}

// A table of the rows for reading, made from them alone: a column for each metric (metrics.views), each part of the
// metadata (meta.storyFlag, meta.record.clip.variants.<experiment>) and YouTube's retention (youtube.*). A list, or a
// map with a key a row of its own, is one cell of JSON. The columns are those of all the rows, so a metric the service
// adds is a column the next time the table is made.
func Table(rows []Row) (header []string, records [][]string) {
	cells := make([]map[string]string, len(rows))
	seen := cpcolls.NewSet[string]()
	for i, row := range rows {
		cells[i] = cellsOf(row)
		for column := range cells[i] {
			seen.Add(column)
		}
	}

	header = slices.Clone(fixedColumns)
	var named []string
	seen.ForEach(func(column string) {
		if !slices.Contains(fixedColumns, column) {
			named = append(named, column)
		}
	})
	sort.Strings(named)
	header = append(header, named...)

	records = make([][]string, len(rows))
	for i := range rows {
		record := make([]string, len(header))
		for j, column := range header {
			record[j] = cells[i][column]
		}
		records[i] = record
	}
	return header, records
}

func cellsOf(row Row) map[string]string {
	cells := map[string]string{
		"takenAt": row.TakenAt.UTC().Format(time.RFC3339), "postId": string(row.PostID), "platform": row.Platform,
		"accountId": row.AccountID, "platformPostId": row.PlatformPostID, "url": row.URL,
		"publishedAt": row.PublishedAt.UTC().Format(time.RFC3339),
		"ageHours":    strconv.FormatFloat(row.AgeHours, 'f', -1, 64),
		"httpStatus":  strconv.Itoa(row.Answer.HTTPStatus), "postStatus": row.Answer.Status,
		"syncStatus": row.Answer.SyncStatus,
	}

	var entry map[string]any
	if decode(row.Analytics, &entry) {
		if status, ok := entry["syncStatus"].(string); ok && status != "" {
			cells["syncStatus"] = status
		}
		if message, ok := entry["errorMessage"].(string); ok {
			cells["errorMessage"] = message
		}
		flatten(cells, "metrics", entry["analytics"])
	}

	var metadata any
	if decode(row.Metadata, &metadata) {
		flatten(cells, "meta", metadata)
	}

	if row.Retention != nil {
		retentionCells(cells, *row.Retention)
	}
	return cells
}

func retentionCells(cells map[string]string, retention Retention) {
	cells["youtube.retentionStatus"] = strconv.Itoa(retention.HTTPStatus)
	var said struct {
		DurationSeconds json.Number `json:"durationSeconds"`
		Curve           []struct {
			Relative *float64 `json:"relativeRetentionPerformance"`
		} `json:"retentionCurve"`
	}
	if !decode(retention.Body, &said) {
		return
	}
	cells["youtube.durationSeconds"] = said.DurationSeconds.String()
	if len(said.Curve) == 0 {
		return
	}
	var curve any
	if decode(retention.Body, &curve) {
		cells["youtube.retentionCurve"] = compact(curve.(map[string]any)["retentionCurve"])
	}
	cells["youtube.points"] = strconv.Itoa(len(said.Curve))

	// How it kept its viewers against videos of its length, over the whole video: 0 the worst, 0.5 the median, 1 the best.
	total, measured := 0.0, 0
	for _, point := range said.Curve {
		if point.Relative != nil {
			total += *point.Relative
			measured++
		}
	}
	if measured > 0 {
		cells["youtube.meanRelativeRetentionPerformance"] = strconv.FormatFloat(total/float64(measured), 'f', 4, 64)
	}
}

func flatten(cells map[string]string, prefix string, value any) {
	switch value := value.(type) {
	case nil:
		return
	case map[string]any:
		for key, inner := range value {
			column := prefix + "." + key
			if slices.Contains(wideKeys, key) {
				cells[column] = compact(inner)
				continue
			}
			flatten(cells, column, inner)
		}
	case []any:
		cells[prefix] = compact(value)
	case string:
		cells[prefix] = value
	case json.Number:
		cells[prefix] = value.String()
	case bool:
		cells[prefix] = strconv.FormatBool(value)
	}
}

func compact(value any) string {
	written, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(written))
}

// With the numbers as they were written.
func decode(raw json.RawMessage, into any) bool {
	if len(raw) == 0 {
		return false
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	return decoder.Decode(into) == nil
}
