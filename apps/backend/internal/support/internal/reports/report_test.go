package reports_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
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

func TestAReportIsBuiltFromWhoReportsWhoIsReportedAndTheComplaint(t *testing.T) {
	said := reports.Complaint{Reason: reports.ReasonChatAbuse, Text: "it insults everyone", Message: "message-1"}

	report, err := reports.NewReport(reports.ReportID{15: 1}, ada, bob, said, createdAt, nil)

	require.NoError(t, err)
	assert.Equal(t, reports.Snapshot{
		ID: reports.ReportID{15: 1}, Reporter: ada, Reported: bob, Complaint: said, At: createdAt,
	}, report.TakeSnapshot())
}

func TestASnapshotRestoresTheReportItWasTakenFrom(t *testing.T) {
	report, err := valid().Restore()
	require.NoError(t, err)

	again, err := report.TakeSnapshot().Restore()

	require.NoError(t, err)
	assert.Equal(t, report, again)
	assert.Equal(t, valid(), again.TakeSnapshot())
}

func TestAReportOfOneselfCannotBeBuilt(t *testing.T) {
	_, err := reports.NewReport(reports.ReportID{15: 1}, ada, ada, reports.Complaint{Reason: reports.ReasonCheating}, createdAt, nil)

	require.ErrorIs(t, err, reports.ErrOwnAccount)
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

func TestEveryReasonIsKnownAndNothingElse(t *testing.T) {
	for _, reason := range []reports.Reason{reports.ReasonCheating, reports.ReasonChatAbuse, reports.ReasonBadName, reports.ReasonOther} {
		assert.Equal(t, reason, complaintKept(t, reports.Complaint{Reason: reason}).Reason)
	}
	for _, reason := range []reports.Reason{"", "boring", "CHEATING"} {
		_, err := reports.NewReport(reports.ReportID{15: 1}, ada, bob, reports.Complaint{Reason: reason}, createdAt, nil)

		require.ErrorIs(t, err, reports.ErrUnknownReason, string(reason))
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

func TestAComplaintWithATextTooLongIsRefused(t *testing.T) {
	_, err := reports.NewReport(reports.ReportID{15: 1}, ada, bob,
		reports.Complaint{Reason: reports.ReasonOther, Text: strings.Repeat("a", notes.MaxTextLength+1)}, createdAt, nil)

	require.ErrorIs(t, err, notes.ErrInvalidText)
}

func TestAMessageIDIsCleanedAndOneTooLongIsDropped(t *testing.T) {
	cleaned := complaintKept(t, reports.Complaint{Reason: reports.ReasonChatAbuse, Message: "mess\x00age\n-1"})
	assert.Equal(t, reports.MessageID("message-1"), cleaned.Message)

	dropped := complaintKept(t, reports.Complaint{Reason: reports.ReasonChatAbuse, Message: reports.MessageID(strings.Repeat("a", 65))})
	assert.Equal(t, reports.NoMessage, dropped.Message)
}

func TestAShownNameIsTrimmedAndOneNoPlayerCanShowIsRefused(t *testing.T) {
	name, err := reports.ShownNameOf("  guest_a1b2c3 ")
	require.NoError(t, err)
	assert.Equal(t, reports.ShownName("guest_a1b2c3"), name)

	for _, value := range []string{"", "   ", strings.Repeat("a", 33)} {
		_, err := reports.ShownNameOf(value)

		require.ErrorIs(t, err, reports.ErrNoPlayer, value)
	}
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

func TestOnlyTheSameReporterAboutTheSamePlayerIsARepeat(t *testing.T) {
	eve := reports.AccountID{15: 3}

	require.NoError(t, again(createdAt,
		earlier(t, eve, bob, createdAt), earlier(t, ada, eve, createdAt), earlier(t, bob, ada, createdAt)))
	require.ErrorIs(t, again(createdAt, earlier(t, eve, bob, createdAt), earlier(t, ada, bob, createdAt)),
		reports.ErrReportedAlready)
}

func TestAReportThatBreaksARuleIsRefusedBeforeItIsARepeat(t *testing.T) {
	_, err := reports.NewReport(reports.ReportID{15: 1}, ada, bob, reports.Complaint{Reason: "boring"}, createdAt,
		[]reports.Report{earlier(t, ada, bob, createdAt)})

	require.ErrorIs(t, err, reports.ErrUnknownReason)
}

func TestAForgottenReportSaysSoInItsSnapshotAndKeepsTheRest(t *testing.T) {
	report, err := valid().Restore()
	require.NoError(t, err)

	forgotten := report.Forgotten()

	want := valid()
	want.Forgotten = true
	assert.Equal(t, want, forgotten.TakeSnapshot())
	assert.False(t, report.TakeSnapshot().Forgotten, "the report it was made from is left as it was")
}

func TestEveryReportOfAListIsForgottenInItsOrder(t *testing.T) {
	first, err := valid().Restore()
	require.NoError(t, err)
	other := valid()
	other.ID = reports.ReportID{15: 2}
	second, err := other.Restore()
	require.NoError(t, err)

	forgotten := reports.AllForgotten([]reports.Report{first, second})

	require.Len(t, forgotten, 2)
	assert.Equal(t, first.Forgotten(), forgotten[0])
	assert.Equal(t, second.Forgotten(), forgotten[1])
	assert.Empty(t, reports.AllForgotten(nil))
}
