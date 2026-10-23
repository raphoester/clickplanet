package prune_place_reports_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/places"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/places/usecases/prune_place_reports_usecase"
)

var now = time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)

type stubReports struct {
	kept   []places.Report
	cutoff time.Time
}

func (s *stubReports) WithReporterBefore(_ context.Context, cutoff time.Time, limit int) ([]places.Report, error) {
	s.cutoff = cutoff
	var found []places.Report
	for _, report := range s.kept {
		if snapshot := report.TakeSnapshot(); snapshot.Reporter != (places.AccountID{}) && len(found) < limit {
			found = append(found, report)
		}
	}
	return found, nil
}

func (s *stubReports) Save(_ context.Context, reports ...places.Report) error {
	for _, saved := range reports {
		for i, report := range s.kept {
			if report.TakeSnapshot().ID == saved.TakeSnapshot().ID {
				s.kept[i] = saved
			}
		}
	}
	return nil
}

func TestEveryReportPastTheRetentionLosesItsReporter(t *testing.T) {
	reports := &stubReports{}
	for n := range 3 {
		report, err := places.NewReport(places.ReportID{15: byte(n + 1)}, places.AccountID{15: 1}, 42,
			places.Complaint{Reason: places.ReasonOther}, now.Add(-100*24*time.Hour))
		require.NoError(t, err)
		reports.kept = append(reports.kept, report)
	}

	anonymized, err := prune_place_reports_usecase.New(90*24*time.Hour, cptime.NewFixedClock(now), reports).Execute(t.Context())
	require.NoError(t, err)

	assert.Equal(t, int64(3), anonymized)
	assert.Equal(t, now.Add(-90*24*time.Hour), reports.cutoff)
	for _, report := range reports.kept {
		assert.Equal(t, places.AccountID{}, report.TakeSnapshot().Reporter)
	}
}
