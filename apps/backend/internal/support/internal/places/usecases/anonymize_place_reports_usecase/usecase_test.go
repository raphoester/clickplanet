package anonymize_place_reports_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/places"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/places/usecases/anonymize_place_reports_usecase"
)

var ada = places.AccountID{15: 1}

type stubReports struct {
	filed []places.Report
	saved []places.Snapshot
	err   error
}

func (s *stubReports) FiledBy(context.Context, places.AccountID) ([]places.Report, error) {
	return s.filed, s.err
}

func (s *stubReports) Save(_ context.Context, reports ...places.Report) error {
	for _, report := range reports {
		s.saved = append(s.saved, report.TakeSnapshot())
	}
	return nil
}

func TestTheReportsAnAccountFiledLoseTheirReporterAndStay(t *testing.T) {
	at := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	filed, err := places.NewReport(places.ReportID{15: 1}, ada, 42, places.Complaint{Reason: places.ReasonHate}, at)
	require.NoError(t, err)
	reports := &stubReports{filed: []places.Report{filed}}

	require.NoError(t, anonymize_place_reports_usecase.New(reports).Execute(t.Context(), ada))

	assert.Equal(t, []places.Snapshot{{
		ID: places.ReportID{15: 1}, Tile: 42, Complaint: places.Complaint{Reason: places.ReasonHate}, At: at,
	}}, reports.saved)
}

func TestAFailedReadAnonymizesNothing(t *testing.T) {
	cause := errors.New("postgres is down")
	reports := &stubReports{err: cause}

	require.ErrorIs(t, anonymize_place_reports_usecase.New(reports).Execute(t.Context(), ada), cause)
	assert.Empty(t, reports.saved)
}
