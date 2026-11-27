package appeal_submission_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/appeals"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/appeals/usecases/appeal_submission_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now    = time.Date(2026, 11, 27, 9, 0, 0, 0, time.UTC)
	ada    = catalog.AccountID{15: 1}
	smurfs = submissions.SubmissionID{15: 1}
)

type fixture struct {
	submissions *submissions.MemoryStore
	judged      *judgements.MemoryStore
	appeals     *appeals.MemoryStore
	outbox      *cpatomic.FakeOutbox
	useCase     *appeal_submission_usecase.UseCase
}

func newFixture(t *testing.T, verdict judgements.Verdict) fixture {
	t.Helper()
	f := fixture{
		submissions: submissions.NewMemoryStore(), judged: judgements.NewMemoryStore(), appeals: appeals.NewMemoryStore(),
		outbox: &cpatomic.FakeOutbox{},
	}
	row := submissions.TestWaiting(smurfs, ada, now.Add(-time.Hour))
	row.Name, row.State = "Smurfland", submissions.Rejected
	row.Rejection = submissions.Rejection{Reason: submissions.NotAPlace, Note: "A cartoon is not a state."}
	f.submissions.Seed(row)
	judgement, err := judgements.Snapshot{
		Submission: smurfs, Verdict: verdict, Why: "a cartoon's village", JudgedAt: now.Add(-time.Hour),
	}.Restore()
	require.NoError(t, err)
	require.NoError(t, f.judged.Save(t.Context(), judgement))
	f.useCase = appeal_submission_usecase.New(cptime.NewFixedClock(now), cpatomic.FakeTransactioner{},
		appeal_submission_usecase.Stores{Submissions: f.submissions, Judgements: f.judged, Appeals: f.appeals}, f.outbox)
	return f
}

func asked(account catalog.AccountID) appeal_submission_usecase.In {
	return appeal_submission_usecase.In{Account: account.String(), Linked: true, Submission: smurfs.String()}
}

func TestAFounderAsksAPersonToLookAgainAtTheJudgesRejection(t *testing.T) {
	f := newFixture(t, judgements.Reject)

	out, err := f.useCase.Execute(t.Context(), asked(ada))
	require.NoError(t, err)

	assert.Equal(t, submissions.Waiting, out.Submission.State)
	assert.Equal(t, appeals.Asked, out.Appeal.State)
	kept, err := f.submissions.Submission(t.Context(), smurfs)
	require.NoError(t, err)
	assert.Equal(t, submissions.Waiting, kept.TakeSnapshot().State)
	assert.Equal(t, appeals.Asked, f.appeals.Kept()[smurfs].State)
	assert.Equal(t, judgements.Reject, f.judged.Kept()[smurfs].Verdict, "never judged again: its judgement stays")
	require.Len(t, f.outbox.Recorded(), 1)
	told, ok := f.outbox.Recorded()[0].(*flagsv1.ReviewAsked)
	require.True(t, ok)
	assert.Equal(t, flagsv1.ReviewCause_REVIEW_CAUSE_APPEAL, told.GetCause())
	assert.Equal(t, "Smurfland", told.GetName())
	assert.Equal(t, "a cartoon's village", told.GetWhy())

	_, err = f.useCase.Execute(t.Context(), asked(ada))
	require.ErrorIs(t, err, appeals.ErrAskedAlready, "once per submission")
}

func TestAnAppealIsRefusedToAGuestAStrangerAndAnOperatorsRejection(t *testing.T) {
	f := newFixture(t, judgements.Unsure)

	_, err := f.useCase.Execute(t.Context(), appeal_submission_usecase.In{
		Account: ada.String(), Submission: smurfs.String(),
	})
	require.ErrorIs(t, err, submissions.ErrGuest)
	_, err = f.useCase.Execute(t.Context(), asked(ada))
	require.ErrorIs(t, err, appeals.ErrNotTheJudges)

	f = newFixture(t, judgements.Reject)
	_, err = f.useCase.Execute(t.Context(), asked(catalog.AccountID{15: 9}))
	require.ErrorIs(t, err, submissions.ErrNotYours)
	assert.Empty(t, f.appeals.Kept())
	assert.Empty(t, f.outbox.Recorded())
}
