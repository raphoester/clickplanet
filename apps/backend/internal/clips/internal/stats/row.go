package stats

import (
	"encoding/json"
	"math"
	"time"
)

// Which shape a row is: a change that is not additive is a new version.
const RowVersion = 1

// One platform of one post, as the service measured it at one snapshot. The rows of the snapshots are the source of
// truth: what the service answered is kept as it came, and a table is made from them.
type Row struct {
	Version int       `json:"version"`
	TakenAt time.Time `json:"takenAt"`

	PostID         PostID    `json:"postId"`
	Platform       string    `json:"platform"`
	AccountID      string    `json:"accountId"`
	PlatformPostID string    `json:"platformPostId"`
	URL            string    `json:"url"`
	PublishedAt    time.Time `json:"publishedAt"`
	// How long it had been out when the snapshot was taken.
	AgeHours float64 `json:"ageHours"`

	// The post's answer, of which this platform's entry is a part.
	Answer Answer `json:"answer"`
	// The platform's entry of the answer as it came, every metric in it. Null when the answer had none for it.
	Analytics json.RawMessage `json:"analytics"`
	// YouTube's alone: the newest curve held, read at this snapshot or carried from the last (its readAt says which),
	// and why the one due at this snapshot was not read, if it was not.
	Retention        *Retention `json:"retention,omitempty"`
	RetentionMissing string     `json:"retentionMissing,omitempty"`
	// The post's metadata as it came: its story key, its countries, its call to act and its record.
	Metadata json.RawMessage `json:"metadata"`
}

// A row for each of the post's targets that is out. retentions: by the target's platform post id.
func RowsOf(post Post, answer Answer, retentions map[string]RetentionState, takenAt time.Time) []Row {
	rows := make([]Row, 0, len(post.Targets))
	for _, target := range post.Targets {
		if !target.Out() {
			continue
		}
		analytics := answer.entryOf(target)
		if analytics == nil {
			analytics = json.RawMessage("null")
		}
		metadata := post.Metadata
		if metadata == nil {
			metadata = json.RawMessage("null")
		}
		retention := retentions[target.PlatformPostID]
		rows = append(rows, Row{
			Version: RowVersion, TakenAt: takenAt.UTC(),
			PostID: post.ID, Platform: target.Platform, AccountID: target.AccountID,
			PlatformPostID: target.PlatformPostID, URL: target.URL, PublishedAt: target.PublishedAt.UTC(),
			AgeHours: ageOf(target.PublishedAt, takenAt),
			Answer:   answer, Analytics: analytics, Metadata: metadata,
			Retention: retention.Retention, RetentionMissing: retention.Missing,
		})
	}
	return rows
}

// Whether its curve was read at this snapshot, not carried from an earlier one.
func (r Row) RetentionRead() bool {
	return r.Retention != nil && r.Retention.ReadAt.Equal(r.TakenAt)
}

// In hours, to the hundredth.
func ageOf(publishedAt, takenAt time.Time) float64 {
	return math.Round(takenAt.Sub(publishedAt).Hours()*100) / 100
}

// Whether the row holds a metric only the TikTok for Business lane gives: TikTok's completion rate, where its views came
// from, its viewers' countries, its retention curve, the profile views and the watch time it brought.
func (r Row) BusinessMetrics() bool {
	var entry struct {
		Analytics *struct {
			CompletionRate    float64            `json:"completionRate"`
			ProfileViews      float64            `json:"profileViews"`
			WatchTime         float64            `json:"igReelsAvgWatchTime"`
			ImpressionSources map[string]float64 `json:"impressionSources"`
			AudienceCountries map[string]float64 `json:"audienceCountries"`
			RetentionCurve    map[string]float64 `json:"retentionCurve"`
		} `json:"analytics"`
	}
	if json.Unmarshal(r.Analytics, &entry) != nil || entry.Analytics == nil {
		return false
	}
	metrics := entry.Analytics
	return metrics.CompletionRate > 0 || metrics.ProfileViews > 0 || metrics.WatchTime > 0 ||
		len(metrics.ImpressionSources) > 0 || len(metrics.AudienceCountries) > 0 || len(metrics.RetentionCurve) > 0
}

// The posts a snapshot reads: those that went out in the last 4 weeks. TikTok's business metrics come only for a post
// active in the last 7 days, and a Short's views have about settled by then; the backfill asks for more.
const window = 28 * day

const day = 24 * time.Hour

func DefaultSince(now time.Time) time.Time {
	return now.Add(-window)
}

// Whether the service is still fetching the platform's numbers.
func syncing(row Row) bool {
	var entry struct {
		SyncStatus string `json:"syncStatus"`
	}
	if json.Unmarshal(row.Analytics, &entry) == nil && entry.SyncStatus != "" {
		return entry.SyncStatus == "pending"
	}
	return row.Answer.SyncStatus == "pending"
}
