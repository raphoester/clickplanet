package stats

import "time"

// What a snapshot read and kept: counts and states, never a metric. The run's summary is public.
type Report struct {
	TakenAt time.Time
	Since   time.Time
	// The posts of the job read, and those of them not out on any platform yet.
	Posts  int
	NotOut int
	// The rows kept, by platform.
	Rows map[string]int
	// Rows of a platform whose numbers the service is still fetching.
	Syncing int
	// The YouTube rows with a retention curve, read now or carried from the last snapshot; the curves asked of
	// YouTube; why those read now came without one; and why the curves due were not read, by reason. A curve missing
	// loses nothing else of its row.
	Curves           int
	RetentionAsked   int
	CurvesRefused    map[string]int
	RetentionMissing map[string]int
	TikTokAccounts   []TikTokLane
	// What could not be read, with why: the snapshot of those is lost.
	Unread []Unread
}

type TikTokLane struct {
	TikTokAccount
	// Its rows of this snapshot, and how many of them hold a metric only the business lane gives.
	Rows         int
	BusinessRows int
}

// Live when its lane answers: the commercial music library, or a business metric on one of its posts. Not its Type,
// which is what TikTok said when the account was connected: on 2026-10-24 an account Zernio still called personal
// answered both, with business metrics on 80 of its rows.
func (l TikTokLane) Live() bool {
	return l.CommercialMusic || l.BusinessRows > 0
}

type Unread struct {
	PostID PostID
	Reason string
}

// Whether every post out was read whole.
func (r Report) Complete() bool {
	return len(r.Unread) == 0
}

func (r Report) RowCount() int {
	total := 0
	for _, count := range r.Rows {
		total += count
	}
	return total
}

func ReportOf(accounts Accounts, rows []Row) Report {
	report := Report{Rows: map[string]int{}, CurvesRefused: map[string]int{}, RetentionMissing: map[string]int{}}
	for _, row := range rows {
		report.Rows[row.Platform]++
		if syncing(row) {
			report.Syncing++
		}
		if row.Retention.Curve() {
			report.Curves++
		} else if row.RetentionRead() {
			report.CurvesRefused[row.Retention.Why()]++
		}
		if row.RetentionMissing != "" {
			report.RetentionMissing[row.RetentionMissing]++
		}
	}
	for _, account := range accounts.TikTok {
		lane := TikTokLane{TikTokAccount: account}
		for _, row := range rows {
			if row.Platform == "tiktok" && row.AccountID == account.ID {
				lane.Rows++
				if row.BusinessMetrics() {
					lane.BusinessRows++
				}
			}
		}
		report.TikTokAccounts = append(report.TikTokAccounts, lane)
	}
	return report
}
