package stats

import (
	"errors"
	"time"
)

var retentionAges = []time.Duration{2 * day, 7 * day}

var ErrRetentionQuota = errors.New("youtube's daily analytics quota is spent")

const QuotaSpent = "youtube's daily analytics quota is spent"

type RetentionState struct {
	Retention *Retention
	Missing   string
}

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

func (r Retention) ReadOn(target Target, takenAt time.Time) *Retention {
	r.ReadAt, r.AgeHours = takenAt.UTC(), ageOf(target.PublishedAt, takenAt)
	return &r
}

func RetentionAfter(held *Retention, read *Retention) RetentionState {
	if read.Curve() || !held.Curve() {
		return RetentionState{Retention: read}
	}
	return RetentionState{Retention: held, Missing: read.Why()}
}

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
