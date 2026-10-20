package report_place_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/places"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/places/usecases/report_place_usecase"
)

var (
	now = time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	ada = places.AccountID{15: 1}
)

type sequentialIDs struct{ next byte }

func (s *sequentialIDs) NewID() (places.ReportID, error) {
	s.next++
	return places.ReportID{15: s.next}, nil
}

type failingIDs struct{}

func (failingIDs) NewID() (places.ReportID, error) {
	return places.ReportID{}, errors.New("no entropy")
}

type recorded struct {
	saved []places.Snapshot
	err   error
}

func (r *recorded) Save(_ context.Context, reports ...places.Report) error {
	if r.err != nil {
		return r.err
	}
	for _, report := range reports {
		r.saved = append(r.saved, report.TakeSnapshot())
	}
	return nil
}

func report(ids places.IDProvider, saver *recorded, in report_place_usecase.In) error {
	return report_place_usecase.New(ids, cptime.NewFixedClock(now), saver).Execute(context.Background(), in) //nolint:wrapcheck // the tests read the sentinel.
}

func TestAReportIsKeptWithAnIDAndTheTimeItCameIn(t *testing.T) {
	saver := &recorded{}

	require.NoError(t, report(&sequentialIDs{}, saver, report_place_usecase.In{
		Reporter: ada, Tile: 42, Complaint: places.Complaint{Reason: places.ReasonHate, Text: "a swastika"},
	}))

	assert.Equal(t, []places.Snapshot{{
		ID: places.ReportID{15: 1}, Reporter: ada, Tile: 42,
		Complaint: places.Complaint{Reason: places.ReasonHate, Text: "a swastika"}, At: now,
	}}, saver.saved)
}

func TestAReportThatBreaksARuleIsNotKept(t *testing.T) {
	saver := &recorded{}

	err := report(&sequentialIDs{}, saver, report_place_usecase.In{Reporter: ada, Tile: 42, Complaint: places.Complaint{Reason: "ugly"}})

	require.ErrorIs(t, err, places.ErrUnknownReason)
	assert.Empty(t, saver.saved)
}

func TestAReportWithNoIDOrNoStoreFails(t *testing.T) {
	in := report_place_usecase.In{Reporter: ada, Tile: 42, Complaint: places.Complaint{Reason: places.ReasonOther}}

	require.Error(t, report(failingIDs{}, &recorded{}, in))

	cause := errors.New("postgres is down")
	require.ErrorIs(t, report(&sequentialIDs{}, &recorded{err: cause}, in), cause)
}
