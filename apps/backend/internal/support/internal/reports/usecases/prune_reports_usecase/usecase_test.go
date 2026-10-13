package prune_reports_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports/inmemory_report_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports/usecases/prune_reports_usecase"
)

var now = time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)

func report(n int, at time.Time) reports.Snapshot {
	return reports.Snapshot{
		ID: reports.ReportID{14: byte(n >> 8), 15: byte(n)}, Reporter: reports.AccountID{15: 1}, Reported: reports.AccountID{15: 2},
		Complaint: reports.Complaint{Reason: reports.ReasonCheating}, At: at,
	}
}

func TestAPruneForgetsTheReportsPastRetentionAndKeepsTheOthers(t *testing.T) {
	kept := inmemory_report_storage.New()
	fresh := report(2, now.Add(-time.Hour))
	atTheLine := report(3, now.Add(-24*time.Hour))
	kept.Seed(report(1, now.Add(-48*time.Hour)), fresh, atTheLine)

	forgotten, err := prune_reports_usecase.New(24*time.Hour, cptime.NewFixedClock(now), kept).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, int64(1), forgotten)
	assert.Equal(t, []reports.Snapshot{atTheLine, fresh}, kept.Kept())
}

func TestAPruneForgetsMoreReportsThanOneBatchHolds(t *testing.T) {
	kept := inmemory_report_storage.New()
	for n := range 1200 {
		kept.Seed(report(n+1, now.Add(-48*time.Hour).Add(time.Duration(n)*time.Second)))
	}
	fresh := report(2000, now.Add(-time.Hour))
	kept.Seed(fresh)

	forgotten, err := prune_reports_usecase.New(24*time.Hour, cptime.NewFixedClock(now), kept).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, int64(1200), forgotten)
	assert.Equal(t, []reports.Snapshot{fresh}, kept.Kept())
}

func TestAFailedPruneSaysSo(t *testing.T) {
	kept := inmemory_report_storage.New()
	kept.FailWith(errors.New("postgres is down"))

	_, err := prune_reports_usecase.New(time.Hour, cptime.NewFixedClock(now), kept).Execute(t.Context())

	require.Error(t, err)
}
