package reports_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
)

var (
	ada       = reports.AccountID{15: 1}
	bob       = reports.AccountID{15: 2}
	createdAt = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
)

func valid() reports.Snapshot {
	return reports.Snapshot{
		ID: reports.ReportID{15: 1}, Reporter: ada, Reported: bob,
		Complaint: reports.Complaint{Reason: reports.ReasonCheating}, At: createdAt,
	}
}

func complaintKept(t *testing.T, said reports.Complaint) reports.Complaint {
	t.Helper()

	report, err := reports.NewReport(reports.ReportID{15: 1}, ada, bob, said, createdAt, nil)
	require.NoError(t, err)
	return report.TakeSnapshot().Complaint
}

func TestASnapshotThatBreaksARuleRestoresNothing(t *testing.T) {
	for name, broken := range map[string]func(*reports.Snapshot){
		"no id":              func(s *reports.Snapshot) { s.ID = reports.ReportID{} },
		"no reporter":        func(s *reports.Snapshot) { s.Reporter = reports.AccountID{} },
		"no player reported": func(s *reports.Snapshot) { s.Reported = reports.AccountID{} },
		"no time":            func(s *reports.Snapshot) { s.At = time.Time{} },
	} {
		snapshot := valid()
		broken(&snapshot)

		_, err := snapshot.Restore()

		require.ErrorIs(t, err, reports.ErrInvalidReport, name)
	}
}

func TestAComplaintIsKeptWithItsTextCleaned(t *testing.T) {
	kept := complaintKept(t, reports.Complaint{Reason: reports.ReasonChatAbuse, Text: "  it insults\r\neveryone  ", Message: " message-1 "})

	assert.Equal(t, reports.Complaint{Reason: reports.ReasonChatAbuse, Text: "it insults\neveryone", Message: "message-1"}, kept)
}

func TestAComplaintNeedsNeitherATextNorAMessage(t *testing.T) {
	kept := complaintKept(t, reports.Complaint{Reason: reports.ReasonCheating, Text: " \n "})

	assert.Equal(t, reports.Complaint{Reason: reports.ReasonCheating, Message: reports.NoMessage}, kept)
}

func TestAMessageIDIsCleanedAndOneTooLongIsDropped(t *testing.T) {
	cleaned := complaintKept(t, reports.Complaint{Reason: reports.ReasonChatAbuse, Message: "mess\x00age\n-1"})
	assert.Equal(t, reports.MessageID("message-1"), cleaned.Message)

	dropped := complaintKept(t, reports.Complaint{Reason: reports.ReasonChatAbuse, Message: reports.MessageID(strings.Repeat("a", 65))})
	assert.Equal(t, reports.NoMessage, dropped.Message)
}

func earlier(t *testing.T, reporter reports.AccountID, about reports.AccountID, at time.Time) reports.Report {
	t.Helper()

	report, err := reports.Snapshot{
		ID: reports.ReportID{15: 9}, Reporter: reporter, Reported: about,
		Complaint: reports.Complaint{Reason: reports.ReasonOther}, At: at,
	}.Restore()
	require.NoError(t, err)
	return report
}

func again(at time.Time, earlier ...reports.Report) error {
	_, err := reports.NewReport(reports.ReportID{15: 1}, ada, bob, reports.Complaint{Reason: reports.ReasonCheating}, at, earlier)
	return err //nolint:wrapcheck // the tests read the domain's own error.
}

func TestAnAccountReportsAPlayerOnceAUTCDay(t *testing.T) {
	sofia := time.FixedZone("EEST", 3*3600)
	morning := earlier(t, ada, bob, time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC))

	require.ErrorIs(t, again(time.Date(2026, 10, 11, 23, 59, 0, 0, time.UTC), morning), reports.ErrReportedAlready)
	require.ErrorIs(t, again(time.Date(2026, 10, 12, 1, 30, 0, 0, sofia), morning), reports.ErrReportedAlready,
		"the day is the UTC day, whatever the zone")

	require.NoError(t, again(time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC), morning), "the next day")
	require.NoError(t, again(time.Date(2026, 10, 10, 23, 59, 0, 0, time.UTC), morning), "the day before")
}

func TestAReportThatBreaksARuleIsRefusedBeforeItIsARepeat(t *testing.T) {
	_, err := reports.NewReport(reports.ReportID{15: 1}, ada, bob, reports.Complaint{Reason: "boring"}, createdAt,
		[]reports.Report{earlier(t, ada, bob, createdAt)})

	require.ErrorIs(t, err, reports.ErrUnknownReason)
}

func TestAReportIDIsAUUIDAndNothingElse(t *testing.T) {
	id, err := reports.ReportIDOf("00000000-0000-0000-0000-000000000001")
	require.NoError(t, err)
	assert.Equal(t, reports.ReportID{15: 1}, id)

	for _, value := range []string{"", "not-a-report", "00000000-0000-0000-0000-000000000000"} {
		_, err := reports.ReportIDOf(value)
		require.ErrorIs(t, err, reports.ErrInvalidReportID, value)
	}
}
