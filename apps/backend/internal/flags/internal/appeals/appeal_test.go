package appeals_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/appeals"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
)

var (
	at      = time.Date(2026, 11, 27, 9, 0, 0, 0, time.UTC)
	ada     = catalog.AccountID{15: 1}
	smurfs  = submissions.SubmissionID{15: 1}
	cartoon = submissions.Rejection{Reason: submissions.NotAPlace, Note: "A cartoon is not a state."}
)

func judged(t *testing.T, verdict judgements.Verdict) judgements.Judgement {
	t.Helper()
	judgement, err := judgements.Snapshot{
		Submission: smurfs, Verdict: verdict, Why: "a cartoon's village", JudgedAt: at,
	}.Restore()
	require.NoError(t, err)
	return judgement
}

func rejected(t *testing.T) submissions.Submission {
	t.Helper()
	submission, err := submissions.Restored(submissions.TestWaiting(smurfs, ada, at)).Rejected(cartoon, at)
	require.NoError(t, err)
	return submission
}

func asking(t *testing.T) appeals.Asking {
	t.Helper()
	submission := rejected(t)
	return appeals.Asking{
		Account: ada, Submission: submission, Judgement: judged(t, judgements.Reject),
		Founded: []submissions.Submission{submission},
	}
}

func TestAnAppealOfTheJudgesRejectionPutsItBackForAPerson(t *testing.T) {
	appeal, reopened, facts, err := appeals.Ask(asking(t), at.Add(time.Hour))
	require.NoError(t, err)

	assert.Equal(t, appeals.Snapshot{
		Submission: smurfs, Account: ada, Rejection: cartoon, State: appeals.Asked, AskedAt: at.Add(time.Hour),
		DecidedAt: at.Add(time.Hour),
	}, appeal.TakeSnapshot(), "the rejection it asks about")
	assert.Equal(t, submissions.Waiting, reopened.TakeSnapshot().State)
	assert.True(t, reopened.TakeSnapshot().Appealed)
	require.Len(t, facts, 1)
	assert.True(t, proto.Equal(&flagsv1.ReviewAsked{
		SubmissionId: smurfs.String(), Name: "Rennes", Cause: flagsv1.ReviewCause_REVIEW_CAUSE_APPEAL,
		Why: "a cartoon's village", AskedAt: timestamppb.New(at.Add(time.Hour)),
	}, facts[0]))
}

func TestOnlyARejectionTheJudgeMadeIsAppealed(t *testing.T) {
	for name, judgement := range map[string]judgements.Judgement{
		"the judge was unsure, the operator rejected it": judged(t, judgements.Unsure),
		"the judge never judged it":                      {},
	} {
		asked := asking(t)
		asked.Judgement = judgement
		_, _, _, err := appeals.Ask(asked, at)
		require.ErrorIs(t, err, appeals.ErrNotTheJudges, name)
	}
}

func TestASubmissionIsAppealedOnce(t *testing.T) {
	first, _, _, err := appeals.Ask(asking(t), at)
	require.NoError(t, err)

	again := asking(t)
	again.Earlier = []appeals.Appeal{first}
	_, _, _, err = appeals.Ask(again, at.Add(time.Hour))
	require.ErrorIs(t, err, appeals.ErrAskedAlready, "the operator's rejection after it is final")
}

func TestTheSubmissionsOwnRulesHold(t *testing.T) {
	stranger := asking(t)
	stranger.Account = catalog.AccountID{15: 9}
	_, _, _, err := appeals.Ask(stranger, at)
	require.ErrorIs(t, err, submissions.ErrNotYours)

	another := asking(t)
	another.Founded = append(another.Founded,
		submissions.Restored(submissions.TestWaiting(submissions.SubmissionID{15: 2}, ada, at)))
	_, _, _, err = appeals.Ask(another, at)
	require.ErrorIs(t, err, submissions.ErrWaiting, "one waits at a time")
}

func TestTheOperatorsVerdictAfterTheAskAnswersIt(t *testing.T) {
	appeal, _, _, err := appeals.Ask(asking(t), at.Add(time.Hour))
	require.NoError(t, err)

	_, changed := appeal.Answered(at)
	assert.False(t, changed, "the judge's rejection, heard late")

	answered, changed := appeal.Answered(at.Add(2 * time.Hour))
	require.True(t, changed)
	assert.Equal(t, appeals.Answered, answered.TakeSnapshot().State)
	assert.Equal(t, at.Add(2*time.Hour), answered.TakeSnapshot().DecidedAt)

	_, changed = answered.Answered(at.Add(3 * time.Hour))
	assert.False(t, changed, "told twice")
}
