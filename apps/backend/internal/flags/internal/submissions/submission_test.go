package submissions_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	at     = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	ada    = catalog.AccountID{15: 1}
	first  = submissions.SubmissionID{15: 1}
	second = submissions.SubmissionID{15: 2}
	rennes = submissions.Draft{Name: "Rennes", Pin: catalog.TestPin}
	none   = catalog.NamesOf(nil)
)

func submitted(t *testing.T, draft submissions.Draft, earlier ...submissions.Submission) submissions.Submission {
	t.Helper()
	submission, _, err := submissions.Submit(second, ada, draft, submissions.TestDrawn, earlier, at)
	require.NoError(t, err)
	return submission
}

func TestAGuestFoundsNoFlag(t *testing.T) {
	_, err := submissions.FounderOf(ada.String(), false)
	require.ErrorIs(t, err, submissions.ErrGuest)
	_, err = submissions.FounderOf("", true)
	require.ErrorIs(t, err, submissions.ErrGuest, "a call with no account")

	account, err := submissions.FounderOf(ada.String(), true)
	require.NoError(t, err)
	assert.Equal(t, ada, account)
}

func TestADraftIsANameAPinAndTheSubmissionItReplaces(t *testing.T) {
	draft, err := submissions.ParseDraft(" Rennes ", 48.1173, -1.6778, "")
	require.NoError(t, err)
	assert.Equal(t, rennes, draft)

	again, err := submissions.ParseDraft("Rennes", 48.1173, -1.6778, first.String())
	require.NoError(t, err)
	assert.Equal(t, first, again.Replaces)

	_, err = submissions.ParseDraft("", 0, 0, "")
	require.ErrorIs(t, err, catalog.ErrInvalidName)
	_, err = submissions.ParseDraft("Rennes", 90.5, 0, "")
	require.ErrorIs(t, err, catalog.ErrInvalidPin)
	_, err = submissions.ParseDraft("Rennes", 0, -180.5, "")
	require.ErrorIs(t, err, catalog.ErrInvalidPin)
	_, err = submissions.ParseDraft("Rennes", 0, 0, "fr")
	require.ErrorIs(t, err, submissions.ErrUnknownSubmission)
}

func TestASubmissionWaitsWithItsPictureHeldInPrivate(t *testing.T) {
	submission := submitted(t, rennes)

	assert.Equal(t, submissions.Snapshot{
		ID: second, Account: ada, Name: "Rennes", Pin: catalog.TestPin,
		Picture: catalog.TestPicture, OtherColors: 3, Held: true, State: submissions.Waiting, SubmittedAt: at,
		DecidedAt: at,
	}, submission.TakeSnapshot())
}

func TestOneSubmissionOfAnAccountWaitsAtATime(t *testing.T) {
	waiting := submissions.Restored(submissions.TestWaiting(first, ada, at))

	_, _, err := submissions.Submit(second, ada, rennes, submissions.TestDrawn, []submissions.Submission{waiting}, at)
	require.ErrorIs(t, err, submissions.ErrWaiting)

	rejected, err := waiting.Rejected(submissions.Rejection{Reason: submissions.Exists}, at)
	require.NoError(t, err)
	submitted(t, rennes, rejected)
}

func TestASubmissionReplacesTheOneThatWaitsNeverOneDecided(t *testing.T) {
	waiting := submissions.Restored(submissions.TestWaiting(first, ada, at))
	again := submissions.Draft{Name: "Breizh", Pin: catalog.TestPin, Replaces: first}

	submission, replaced, err := submissions.Submit(second, ada, again, submissions.TestDrawn,
		[]submissions.Submission{waiting}, at)
	require.NoError(t, err)
	assert.Equal(t, []submissions.SubmissionID{first}, replaced)
	assert.Equal(t, submissions.Waiting, submission.TakeSnapshot().State)
	assert.Equal(t, catalog.Name("Breizh"), submission.TakeSnapshot().Name)

	_, replaced, err = submissions.Submit(second, ada, rennes, submissions.TestDrawn, nil, at)
	require.NoError(t, err)
	assert.Empty(t, replaced, "none to replace")

	approved, err := waiting.Approved(none, at)
	require.NoError(t, err)
	_, _, err = submissions.Submit(second, ada, again, submissions.TestDrawn, []submissions.Submission{approved}, at)
	require.ErrorIs(t, err, submissions.ErrNotWaiting, "an approved one never changes")
	_, _, err = submissions.Submit(second, ada, again, submissions.TestDrawn, nil, at)
	require.ErrorIs(t, err, submissions.ErrNotWaiting, "another account's, or one gone")
}

func TestASubmissionIsDecidedOnce(t *testing.T) {
	waiting := submitted(t, rennes)
	approved, err := waiting.Approved(none, at.Add(time.Hour))
	require.NoError(t, err)
	assert.False(t, approved.TakeSnapshot().Held, "public, so no longer held in private")
	assert.Equal(t, at.Add(time.Hour), approved.TakeSnapshot().DecidedAt)

	_, err = approved.Approved(none, at)
	require.ErrorIs(t, err, submissions.ErrDecided)
	_, err = approved.Rejected(submissions.Rejection{Reason: submissions.Hateful}, at)
	require.ErrorIs(t, err, submissions.ErrDecided)

	rejected, err := waiting.Rejected(submissions.Rejection{Reason: submissions.Hateful}, at)
	require.NoError(t, err)
	assert.True(t, rejected.TakeSnapshot().Held, "held until it is deleted")
	_, err = rejected.Approved(none, at)
	require.ErrorIs(t, err, submissions.ErrDecided)
}

func TestARejectionHasAReasonAndOtherNeedsANote(t *testing.T) {
	rejection, err := submissions.ParseRejection(submissions.Other, "  A city: send its region's flag.  ")
	require.NoError(t, err)
	assert.Equal(t, submissions.Rejection{Reason: submissions.Other, Note: "A city: send its region's flag."}, rejection)
	_, err = submissions.ParseRejection(submissions.Unreadable, "")
	require.NoError(t, err)

	for _, invalid := range []submissions.Rejection{
		{Reason: ""},
		{Reason: "rude"},
		{Reason: submissions.Other, Note: "  "},
		{Reason: submissions.Exists, Note: strings.Repeat("é", 501)},
	} {
		_, err := submissions.ParseRejection(invalid.Reason, invalid.Note)
		require.ErrorIs(t, err, submissions.ErrInvalidRejection, "%+v", invalid)
	}
}

func TestAnApprovedNewSubmissionOpensItsFlagAtItsPin(t *testing.T) {
	approved, err := submissions.Restored(submissions.TestWaiting(first, ada, at)).Approved(none, at)
	require.NoError(t, err)

	opened, flag, err := approved.Open(catalog.NewOpening(&catalog.Sequence{}, cptime.NewFixedClock(at.Add(time.Hour))), none,
		at.Add(30*24*time.Hour))
	require.NoError(t, err)

	assert.Equal(t, catalog.Snapshot{
		ID: "r-0000000001", Name: "Rennes", Picture: catalog.TestPicture, Pin: catalog.TestPin, State: catalog.Open,
		DecidedAt: at.Add(time.Hour), EndsAt: at.Add(30 * 24 * time.Hour), Holder: ada,
	}, flag.TakeSnapshot(), "held by its founder")
	assert.Equal(t, submissions.Opened, opened.TakeSnapshot().State)
	assert.Equal(t, catalog.FlagID("r-0000000001"), opened.TakeSnapshot().Flag)

	_, _, err = opened.Open(catalog.NewOpening(&catalog.Sequence{}, cptime.NewFixedClock(at)), none, at.Add(time.Hour))
	require.ErrorIs(t, err, submissions.ErrNotApproved, "once")
}

func TestOnlyAnApprovedSubmissionOpensAFlag(t *testing.T) {
	opening := catalog.NewOpening(&catalog.Sequence{}, cptime.NewFixedClock(at))
	waiting := submissions.Restored(submissions.TestWaiting(first, ada, at))
	_, _, err := waiting.Open(opening, none, at.Add(time.Hour))
	require.ErrorIs(t, err, submissions.ErrNotApproved)

	approved, err := waiting.Approved(none, at)
	require.NoError(t, err)
	_, _, err = approved.Open(opening, none, at)
	require.ErrorIs(t, err, catalog.ErrEnded)
}

func TestARejectedPictureIsDeleted30DaysLater(t *testing.T) {
	rejected, err := submissions.Restored(submissions.TestWaiting(first, ada, at)).
		Rejected(submissions.Rejection{Reason: submissions.NotAPlace}, at)
	require.NoError(t, err)

	_, due := rejected.PictureDeleted(at.Add(30*24*time.Hour - time.Second))
	assert.False(t, due)
	deleted, due := rejected.PictureDeleted(at.Add(30 * 24 * time.Hour))
	require.True(t, due)
	assert.False(t, deleted.TakeSnapshot().Held)
	_, due = deleted.PictureDeleted(at.Add(31 * 24 * time.Hour))
	assert.False(t, due, "once")

	_, due = submissions.Restored(submissions.TestWaiting(first, ada, at)).PictureDeleted(at.Add(60 * 24 * time.Hour))
	assert.False(t, due, "a waiting picture stays until its review")
}

func TestASubmissionKeptInAStateItCannotBeInIsRefused(t *testing.T) {
	for name, edit := range map[string]func(*submissions.Snapshot){
		"no account":                  func(s *submissions.Snapshot) { s.Account = catalog.AccountID{} },
		"no state":                    func(s *submissions.Snapshot) { s.State = "lost" },
		"a flag before its opening":   func(s *submissions.Snapshot) { s.Flag = "r-0000000001" },
		"rejected with no reason":     func(s *submissions.Snapshot) { s.State = submissions.Rejected },
		"a rejection while it waits":  func(s *submissions.Snapshot) { s.Rejection.Reason = submissions.Hateful },
		"held once approved":          func(s *submissions.Snapshot) { s.State = submissions.Approved },
		"opened with no flag":         func(s *submissions.Snapshot) { s.State, s.Held = submissions.Opened, false },
		"a pin off the globe":         func(s *submissions.Snapshot) { s.Pin.Latitude = 91 },
		"a name too long for a flag":  func(s *submissions.Snapshot) { s.Name = catalog.Name(strings.Repeat("a", 41)) },
		"a picture with no good name": func(s *submissions.Snapshot) { s.Picture.Hash = "x" },
	} {
		snapshot := submissions.TestWaiting(first, ada, at)
		edit(&snapshot)
		_, err := snapshot.Restore()
		require.Error(t, err, name)
	}
}

func TestTwoFlagsNeverShareAName(t *testing.T) {
	taken := catalog.NamesOf([]catalog.Flag{catalog.OpenFlag("r-0000000001", "Rennes", at)})
	waiting := submissions.Restored(submissions.TestWaiting(first, ada, at))

	_, err := waiting.Approved(taken, at)
	require.ErrorIs(t, err, catalog.ErrNameTaken, "another flag's name")
	builtIn := submissions.TestWaiting(first, ada, at)
	builtIn.Name = "germany"
	_, err = submissions.Restored(builtIn).Approved(none, at)
	require.ErrorIs(t, err, catalog.ErrNameTaken, "a flag the game ships with, whatever its case")

	approved, err := waiting.Approved(none, at)
	require.NoError(t, err)
	_, _, err = approved.Open(catalog.NewOpening(&catalog.Sequence{}, cptime.NewFixedClock(at)), taken, at.Add(time.Hour))
	require.ErrorIs(t, err, catalog.ErrNameTaken, "a flag took its name since")
}

func TestOnlyAWaitingSubmissionsCapitalMoves(t *testing.T) {
	waiting := submissions.Restored(submissions.TestWaiting(first, ada, at))
	nantes := catalog.Pin{Latitude: 47.218, Longitude: -1.5536}

	moved, err := waiting.Moved(nantes)
	require.NoError(t, err)
	assert.Equal(t, nantes, moved.TakeSnapshot().Pin)

	approved, err := moved.Approved(none, at)
	require.NoError(t, err)
	_, err = approved.Moved(catalog.TestPin)
	require.ErrorIs(t, err, submissions.ErrDecided)
}

func paidFor(snapshot submissions.Snapshot) submissions.Submission {
	snapshot.Paid = true
	return submissions.Restored(snapshot)
}

func TestAFlagPaidForWhileItWaitsIsNotChanged(t *testing.T) {
	again := submissions.Draft{Name: "Breizh", Pin: catalog.TestPin, Replaces: first}

	_, _, err := submissions.Submit(second, ada, again, submissions.TestDrawn,
		[]submissions.Submission{paidFor(submissions.TestWaiting(first, ada, at))}, at)

	require.ErrorIs(t, err, submissions.ErrPaid)
}

func TestItsFounderPaysForAFlagUntilItsTimeIsPaidThenForItsFlag(t *testing.T) {
	waiting := submissions.TestWaiting(first, ada, at)
	approved := waiting
	approved.State, approved.Held = submissions.Approved, false
	opened := approved
	opened.State, opened.Flag = submissions.Opened, "r-0000000001"
	rejected := waiting
	rejected.State, rejected.Rejection = submissions.Rejected, submissions.Rejection{Reason: submissions.Hateful}

	require.NoError(t, submissions.Restored(waiting).PayableBy(ada), "paid before its review")
	require.NoError(t, submissions.Restored(approved).PayableBy(ada), "approved before it was paid for")
	require.NoError(t, submissions.Restored(opened).PayableBy(ada), "time for its flag")
	require.ErrorIs(t, paidFor(waiting).PayableBy(ada), submissions.ErrPaidAlready)
	require.ErrorIs(t, paidFor(approved).PayableBy(ada), submissions.ErrPaidAlready, "it opens with that time")
	require.ErrorIs(t, submissions.Restored(rejected).PayableBy(ada), submissions.ErrNotPayable)
	require.ErrorIs(t, submissions.Restored(waiting).PayableBy(catalog.AccountID{15: 9}), submissions.ErrNotYours)
}

func TestAVerdictIsToldAndNothingElse(t *testing.T) {
	waiting := submitted(t, rennes)
	assert.Empty(t, waiting.Facts())

	approved, err := waiting.Approved(none, at.Add(time.Hour))
	require.NoError(t, err)
	require.Len(t, approved.Facts(), 1)
	assert.Equal(t, second.String(), approved.Facts()[0].(*flagsv1.SubmissionApproved).GetSubmissionId())

	rejected, err := waiting.Rejected(submissions.Rejection{Reason: submissions.Hateful}, at.Add(time.Hour))
	require.NoError(t, err)
	require.Len(t, rejected.Facts(), 1)
	assert.Equal(t, at.Add(time.Hour), rejected.Facts()[0].(*flagsv1.SubmissionRejected).GetRejectedAt().AsTime())
}

func rejectedOf(t *testing.T, snapshot submissions.Snapshot) submissions.Submission {
	t.Helper()
	rejected, err := submissions.Restored(snapshot).Rejected(
		submissions.Rejection{Reason: submissions.NotAPlace, Note: "A club is not a state."}, at)
	require.NoError(t, err)
	return rejected
}

func TestARejectionAppealedWaitsAgainWithNoRejection(t *testing.T) {
	rejected := rejectedOf(t, submissions.TestWaiting(first, ada, at))

	reopened, err := rejected.Reopened(ada, []submissions.Submission{rejected}, at.Add(time.Hour))
	require.NoError(t, err)

	kept := reopened.TakeSnapshot()
	assert.Equal(t, submissions.Waiting, kept.State)
	assert.Equal(t, submissions.Rejection{}, kept.Rejection)
	assert.True(t, kept.Appealed)
	assert.True(t, kept.Held, "the picture stays private")
	assert.Equal(t, at.Add(time.Hour), kept.DecidedAt)
	assert.Empty(t, reopened.Facts(), "nothing is approved nor rejected")
}

func TestOnlyItsFounderReopensARejectionHeldWhileNoOtherWaits(t *testing.T) {
	rejected := rejectedOf(t, submissions.TestWaiting(first, ada, at))
	_, err := rejected.Reopened(catalog.AccountID{15: 9}, nil, at)
	require.ErrorIs(t, err, submissions.ErrNotYours)

	waiting := submissions.Restored(submissions.TestWaiting(second, ada, at))
	_, err = rejected.Reopened(ada, []submissions.Submission{rejected, waiting}, at)
	require.ErrorIs(t, err, submissions.ErrWaiting)

	_, err = waiting.Reopened(ada, nil, at)
	require.ErrorIs(t, err, submissions.ErrNotRejected)

	gone, deleted := rejected.PictureDeleted(at.Add(31 * 24 * time.Hour))
	require.True(t, deleted)
	_, err = gone.Reopened(ada, nil, at.Add(31*24*time.Hour))
	require.ErrorIs(t, err, submissions.ErrPictureGone)
}

func TestASubmissionAPersonLooksAtAgainIsNotReplaced(t *testing.T) {
	appealed := submissions.TestWaiting(first, ada, at)
	appealed.Appealed = true

	_, _, err := submissions.Submit(second, ada, submissions.Draft{Name: "Rennes", Pin: catalog.TestPin, Replaces: first},
		submissions.TestDrawn, []submissions.Submission{submissions.Restored(appealed)}, at)
	require.ErrorIs(t, err, submissions.ErrAppealed)
}

func TestASubmissionPaidForWaitsPaidFor(t *testing.T) {
	waiting := submissions.Restored(submissions.TestWaiting(first, ada, at))
	assert.False(t, waiting.WaitsPaidFor())
	assert.True(t, waiting.PaidFor().WaitsPaidFor())
}
