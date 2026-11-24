package submissions_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	at     = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	ada    = submissions.AccountID{15: 1}
	first  = submissions.SubmissionID{15: 1}
	second = submissions.SubmissionID{15: 2}
	rennes = submissions.Draft{Name: "Rennes", Pin: catalog.TestPin}
)

func submitted(t *testing.T, draft submissions.Draft, earlier ...submissions.Submission) submissions.Submission {
	t.Helper()
	submission, err := submissions.Submit(second, ada, draft, submissions.TestDrawn, earlier, at)
	require.NoError(t, err)
	return submission
}

func opened(t *testing.T) submissions.Submission {
	t.Helper()
	approved, err := submissions.Restored(submissions.TestWaiting(first, ada, at)).Approved(at)
	require.NoError(t, err)
	opened, _, err := approved.Open(catalog.NewOpening(&catalog.Sequence{}, cptime.NewFixedClock(at)), at.Add(time.Hour))
	require.NoError(t, err)
	return opened
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

func TestADraftIsANameAPinAndForAChangeAFlag(t *testing.T) {
	draft, err := submissions.ParseDraft(" Rennes ", 48.1173, -1.6778, "")
	require.NoError(t, err)
	assert.Equal(t, rennes, draft)

	change, err := submissions.ParseDraft("Rennes", 48.1173, -1.6778, "r-0000000001")
	require.NoError(t, err)
	assert.Equal(t, catalog.FlagID("r-0000000001"), change.Flag)

	_, err = submissions.ParseDraft("", 0, 0, "")
	require.ErrorIs(t, err, catalog.ErrInvalidName)
	_, err = submissions.ParseDraft("Rennes", 90.5, 0, "")
	require.ErrorIs(t, err, catalog.ErrInvalidPin)
	_, err = submissions.ParseDraft("Rennes", 0, -180.5, "")
	require.ErrorIs(t, err, catalog.ErrInvalidPin)
	_, err = submissions.ParseDraft("Rennes", 0, 0, "fr")
	require.ErrorIs(t, err, catalog.ErrInvalidFlagID, "a flag the game ships with has no founder")
}

func TestASubmissionWaitsWithItsPictureHeldInPrivate(t *testing.T) {
	submission := submitted(t, rennes)

	assert.Equal(t, submissions.Snapshot{
		ID: second, Account: ada, Kind: submissions.New, Name: "Rennes", Pin: catalog.TestPin,
		Picture: catalog.TestPicture, OtherColors: 3, Held: true, State: submissions.Waiting, SubmittedAt: at,
		DecidedAt: at,
	}, submission.TakeSnapshot())
}

func TestOneSubmissionOfAnAccountWaitsAtATime(t *testing.T) {
	waiting := submissions.Restored(submissions.TestWaiting(first, ada, at))

	_, err := submissions.Submit(second, ada, rennes, submissions.TestDrawn, []submissions.Submission{waiting}, at)
	require.ErrorIs(t, err, submissions.ErrWaiting)

	rejected, err := waiting.Rejected(submissions.Rejection{Reason: submissions.Exists}, at)
	require.NoError(t, err)
	submitted(t, rennes, rejected)
}

func TestOnlyTheAccountThatFoundedAFlagSubmitsItsChange(t *testing.T) {
	change := submissions.Draft{Name: "Breizh", Pin: catalog.TestPin, Flag: "r-0000000001"}

	_, err := submissions.Submit(second, ada, change, submissions.TestDrawn, nil, at)
	require.ErrorIs(t, err, submissions.ErrNotFounder)
	_, err = submissions.Submit(second, ada, submissions.Draft{Name: "Breizh", Pin: catalog.TestPin, Flag: "r-0000000002"},
		submissions.TestDrawn, []submissions.Submission{opened(t)}, at)
	require.ErrorIs(t, err, submissions.ErrNotFounder, "another flag")

	submission := submitted(t, change, opened(t))
	assert.Equal(t, submissions.Change, submission.TakeSnapshot().Kind)
	assert.Equal(t, catalog.FlagID("r-0000000001"), submission.TakeSnapshot().Flag)
}

func TestASubmissionIsDecidedOnce(t *testing.T) {
	waiting := submitted(t, rennes)
	approved, err := waiting.Approved(at.Add(time.Hour))
	require.NoError(t, err)
	assert.False(t, approved.TakeSnapshot().Held, "public, so no longer held in private")
	assert.Equal(t, at.Add(time.Hour), approved.TakeSnapshot().DecidedAt)

	_, err = approved.Approved(at)
	require.ErrorIs(t, err, submissions.ErrDecided)
	_, err = approved.Rejected(submissions.Rejection{Reason: submissions.Hateful}, at)
	require.ErrorIs(t, err, submissions.ErrDecided)

	rejected, err := waiting.Rejected(submissions.Rejection{Reason: submissions.Hateful}, at)
	require.NoError(t, err)
	assert.True(t, rejected.TakeSnapshot().Held, "held until it is deleted")
	_, err = rejected.Approved(at)
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
	approved, err := submissions.Restored(submissions.TestWaiting(first, ada, at)).Approved(at)
	require.NoError(t, err)

	opened, flag, err := approved.Open(catalog.NewOpening(&catalog.Sequence{}, cptime.NewFixedClock(at.Add(time.Hour))),
		at.Add(30*24*time.Hour))
	require.NoError(t, err)

	assert.Equal(t, catalog.Snapshot{
		ID: "r-0000000001", Name: "Rennes", Picture: catalog.TestPicture, Pin: catalog.TestPin, State: catalog.Open,
		DecidedAt: at.Add(time.Hour), EndsAt: at.Add(30 * 24 * time.Hour),
	}, flag.TakeSnapshot())
	assert.Equal(t, submissions.Opened, opened.TakeSnapshot().State)
	assert.Equal(t, catalog.FlagID("r-0000000001"), opened.TakeSnapshot().Flag)

	_, _, err = opened.Open(catalog.NewOpening(&catalog.Sequence{}, cptime.NewFixedClock(at)), at.Add(time.Hour))
	require.ErrorIs(t, err, submissions.ErrNotApproved, "once")
}

func TestOnlyAnApprovedNewSubmissionOpensAFlag(t *testing.T) {
	opening := catalog.NewOpening(&catalog.Sequence{}, cptime.NewFixedClock(at))
	waiting := submissions.Restored(submissions.TestWaiting(first, ada, at))
	_, _, err := waiting.Open(opening, at.Add(time.Hour))
	require.ErrorIs(t, err, submissions.ErrNotApproved)

	change, err := submitted(t, submissions.Draft{Name: "Breizh", Pin: catalog.TestPin, Flag: "r-0000000001"}, opened(t)).
		Approved(at)
	require.NoError(t, err)
	_, _, err = change.Open(opening, at.Add(time.Hour))
	require.ErrorIs(t, err, submissions.ErrNotApproved, "a change opens nothing")

	approved, err := waiting.Approved(at)
	require.NoError(t, err)
	_, _, err = approved.Open(opening, at)
	require.ErrorIs(t, err, catalog.ErrEnded)
}

func TestAnApprovedChangeRestylesItsFlagAndTellsIt(t *testing.T) {
	change, err := submitted(t, submissions.Draft{Name: "Breizh", Pin: catalog.TestPin, Flag: "r-0000000001"}, opened(t)).
		Approved(at.Add(time.Hour))
	require.NoError(t, err)
	flag := catalog.OpenFlag("r-0000000001", "Rennes", at)

	restyled, fact, err := change.Restyle(flag, at.Add(time.Hour))
	require.NoError(t, err)

	assert.Equal(t, catalog.Name("Breizh"), restyled.TakeSnapshot().Name)
	assert.Equal(t, flag.TakeSnapshot().DecidedAt, restyled.TakeSnapshot().DecidedAt, "the same flag, open as it was")
	assert.True(t, proto.Equal(&flagsv1.FlagUpdated{
		Flag: "r-0000000001", Name: "Breizh", UpdatedAt: timestamppb.New(at.Add(time.Hour)),
	}, fact), "%v", fact)

	_, _, err = submissions.Restored(submissions.TestWaiting(first, ada, at)).Restyle(flag, at)
	require.ErrorIs(t, err, submissions.ErrNotApproved)
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
		"no account":                  func(s *submissions.Snapshot) { s.Account = submissions.AccountID{} },
		"no state":                    func(s *submissions.Snapshot) { s.State = "lost" },
		"a change for no flag":        func(s *submissions.Snapshot) { s.Kind = submissions.Change },
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
