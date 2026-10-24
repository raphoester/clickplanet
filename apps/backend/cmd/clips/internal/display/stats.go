package display

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips"
)

const (
	dayLayout = "Mon 2 Jan 2006"
	// What zernio answers for the curve of a channel connected without YouTube's analytics scope.
	scopeMissing = "youtube_analytics_scope_missing"
)

// Whether every post out was read whole. The summary says what was kept, never a metric: a public repo's run summary is
// public. It is added to the summary file, or written to out when none is named.
func Stats(summaryFile string, out io.Writer, report clips.StatsReport) (bool, error) {
	if summaryFile == "" {
		return statsPage(out, report)
	}

	file, err := os.OpenFile(summaryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec // the file the operator of the job named.
	if err != nil {
		return false, fmt.Errorf("failed to open the summary: %w", err)
	}
	defer func() { _ = file.Close() }()

	return statsPage(file, report)
}

func statsPage(page io.Writer, report clips.StatsReport) (bool, error) {
	var text strings.Builder
	fmt.Fprintf(&text, "### Snapshot of %s\n\n", report.TakenAt.UTC().Format(timeLayout))
	fmt.Fprintf(&text, "%d posts of the job since %s, %d of them not out yet. %d rows kept",
		report.Posts, report.Since.UTC().Format(dayLayout), report.NotOut, report.RowCount())
	if platforms := report.Platforms(); len(platforms) > 0 {
		counts := make([]string, 0, len(platforms))
		for _, platform := range platforms {
			counts = append(counts, fmt.Sprintf("%d %s", report.Rows[platform], platform))
		}
		fmt.Fprintf(&text, " (%s)", strings.Join(counts, ", "))
	}
	fmt.Fprintf(&text, ", %d still syncing.\n\n", report.Syncing)

	text.WriteString(tiktokTable(report))
	text.WriteString(curves(report))

	if !report.Complete() {
		text.WriteString("**Posts not read whole**: their snapshot of today is lost.\n\n| Post | Why |\n|---|---|\n")
		for _, unread := range report.Unread {
			fmt.Fprintf(&text, "| %s | %s |\n", cell(string(unread.PostID)), cell(unread.Reason))
		}
		text.WriteString("\n")
	}

	if _, err := io.WriteString(page, text.String()); err != nil {
		return false, fmt.Errorf("failed to write the summary: %w", err)
	}
	return report.Complete(), nil
}

// Whether TikTok's business lane is live on each account: what TikTok called the account when it was connected, whether
// its business metrics came on a post of this snapshot, and whether the commercial music library answers.
func tiktokTable(report clips.StatsReport) string {
	if len(report.TikTokAccounts) == 0 {
		return "No TikTok account is connected.\n\n"
	}
	var text strings.Builder
	text.WriteString("| TikTok account | tiktokAccountType | Rows with business metrics | Commercial music | Business lane |\n")
	text.WriteString("|---|---|---|---|---|\n")
	for _, lane := range report.TikTokAccounts {
		music := "answers"
		if !lane.CommercialMusic {
			music = "refused: " + cell(lane.CommercialMusicRefusal)
		}
		live := "not live"
		if lane.Live() {
			live = "**live**"
		}
		kind := lane.Type
		if kind == "" {
			kind = "not said"
		}
		fmt.Fprintf(&text, "| %s | %s | %d of %d | %s | %s |\n",
			cell(nameOf(lane.Username, lane.ID)), kind, lane.BusinessRows, lane.Rows, music, live)
	}
	text.WriteString("\nTikTok sends its business metrics 24 to 48 hours after a post, and only for posts active in the last 7 days.\n\n")
	return text.String()
}

func curves(report clips.StatsReport) string {
	if report.Curves == 0 && len(report.CurvesRefused) == 0 {
		return ""
	}
	reasons := make([]string, 0, len(report.CurvesRefused))
	for reason, count := range report.CurvesRefused {
		reasons = append(reasons, fmt.Sprintf("%d %s", count, reason))
	}
	sort.Strings(reasons)
	text := fmt.Sprintf("YouTube retention curves: %d", report.Curves)
	if len(reasons) > 0 {
		text += ", none for " + strings.Join(reasons, ", ")
	}
	text += ".\n\n"
	for reason := range report.CurvesRefused {
		if strings.HasSuffix(reason, scopeMissing) {
			text += "**YouTube's analytics scope is missing**: reconnect the YouTube account in Zernio to grant it.\n\n"
			break
		}
	}
	return text
}

func nameOf(username, id string) string {
	if username == "" {
		return id
	}
	return "@" + strings.TrimPrefix(username, "@")
}
