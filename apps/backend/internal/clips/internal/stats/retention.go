package stats

import (
	"errors"
	"time"
)

// YouTube's retention curve is aggregated over the video's whole life to the day asked, so it is only worth reading at
// a few ages: once past 2 days, when YouTube has measured it, and once past a week, when a Short's views have about
// settled. Not every video every day: the curves of 4 weeks of videos, read daily, spent YouTube's analytics quota on
// 2026-10-25.
var retentionAges = []time.Duration{2 * day, 7 * day}

// Zernio's 429 when YouTube's daily analytics quota is spent: no curve is read again before midnight Pacific, so the
// run stops asking.
var ErrRetentionQuota = errors.New("youtube's daily analytics quota is spent")

// Why a curve that was due was not asked, once the quota was spent earlier in the run.
const QuotaSpent = "youtube's daily analytics quota is spent"

// What a row says of its video's curve: the newest one held, read now or carried from the last snapshot, and why the
// one due now was not read, if it was not.
type RetentionState struct {
	Retention *Retention
	Missing   string
}

// Whether the curve of a video this old is worth asking for: it is past an age at which no curve was read yet. A video
// whose curve came empty, was refused or failed is asked again the next day, until one comes.
func RetentionDue(age time.Duration, held *Retention) bool {
	var reached time.Duration
	for _, threshold := range retentionAges {
		if age >= threshold {
			reached = threshold
		}
	}
	if reached == 0 {
		return false
	}
	return !held.Curve() || time.Duration(held.AgeHours*float64(time.Hour)) < reached
}

// The answer as read at a snapshot, of a video out on target: a copy that says when, and how old the video was.
func (r Retention) ReadOn(target Target, takenAt time.Time) *Retention {
	r.ReadAt, r.AgeHours = takenAt.UTC(), ageOf(target.PublishedAt, takenAt)
	return &r
}

// What a read gives the row: the answer, unless it holds no curve where one was held, which stays (an older curve is
// still true to its day) with why the new one did not come.
func RetentionAfter(held *Retention, read *Retention) RetentionState {
	if read.Curve() || !held.Curve() {
		return RetentionState{Retention: read}
	}
	return RetentionState{Retention: held, Missing: read.Why()}
}

// Each video's curve as a snapshot's rows hold it, by its platform post id. A row written before a curve said when it
// was read was read with its row.
func HeldRetentions(rows []Row) map[string]*Retention {
	held := map[string]*Retention{}
	for _, row := range rows {
		if row.Retention == nil || row.PlatformPostID == "" {
			continue
		}
		retention := *row.Retention
		if retention.ReadAt.IsZero() {
			retention.ReadAt, retention.AgeHours = row.TakenAt, row.AgeHours
		}
		held[row.PlatformPostID] = &retention
	}
	return held
}
