package forget_reports_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports/inmemory_report_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports/usecases/forget_reports_usecase"
)

var (
	at  = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	ada = reports.AccountID{15: 1}
	bob = reports.AccountID{15: 2}
	eve = reports.AccountID{15: 3}
)

func TestAForgottenAccountTakesTheReportsItFiledAndTheReportsAboutIt(t *testing.T) {
	store := inmemory_report_storage.New()
	cheating := reports.Complaint{Reason: reports.ReasonCheating}
	kept := reports.Snapshot{ID: reports.ReportID{15: 3}, Reporter: eve, Reported: bob, Complaint: cheating, At: at}
	store.Seed(
		reports.Snapshot{ID: reports.ReportID{15: 1}, Reporter: ada, Reported: bob, Complaint: cheating, At: at},
		reports.Snapshot{ID: reports.ReportID{15: 2}, Reporter: eve, Reported: ada, Complaint: cheating, At: at},
		kept,
	)

	require.NoError(t, forget_reports_usecase.New(store).Execute(t.Context(), ada))

	assert.Equal(t, []reports.Snapshot{kept}, store.Kept())
}

func TestAnAccountNoReportNamesIsForgottenWithNoError(t *testing.T) {
	store := inmemory_report_storage.New()
	kept := reports.Snapshot{ID: reports.ReportID{15: 1}, Reporter: eve, Reported: bob, Complaint: reports.Complaint{Reason: reports.ReasonCheating}, At: at}
	store.Seed(kept)

	require.NoError(t, forget_reports_usecase.New(store).Execute(t.Context(), ada))

	assert.Equal(t, []reports.Snapshot{kept}, store.Kept())
}

func TestAFailureToForgetIsAnError(t *testing.T) {
	store := inmemory_report_storage.New()
	refused := errors.New("the database is down")
	store.FailWith(refused)

	assert.ErrorIs(t, forget_reports_usecase.New(store).Execute(t.Context(), ada), refused)
}
