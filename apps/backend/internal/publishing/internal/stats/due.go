package stats

import "time"

// The snapshot of a day is taken from 08:17 UTC on: just after midnight Pacific in winter, when YouTube's Analytics
// quota, which every retention curve draws on, starts again (a run at 05:41 UTC found it spent, and the curves of Oct
// 25 and 26 were lost). The minute is the clips workflow's, which took it until 2026-11-10.
const (
	dueHour   = 8
	dueMinute = 17
)

// The time from which today's snapshot is due: today's 08:17 UTC, or yesterday's before it.
func DueSince(now time.Time) time.Time {
	now = now.UTC()
	due := time.Date(now.Year(), now.Month(), now.Day(), dueHour, dueMinute, 0, 0, time.UTC)
	if now.Before(due) {
		return due.AddDate(0, 0, -1)
	}
	return due
}
