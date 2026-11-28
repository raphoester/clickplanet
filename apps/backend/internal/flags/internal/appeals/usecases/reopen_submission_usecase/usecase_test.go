package reopen_submission_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/appeals"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/appeals/usecases/reopen_submission_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now    = time.Date(2026, 11, 28, 9, 0, 0, 0, time.UTC)
	ada    = catalog.AccountID{15: 1}
	smurfs = submissions.SubmissionID{15: 1}
)

type fixture struct {
	submissions *submissions.MemoryStore
	judged      *judgements.MemoryStore
	appeals     *appeals.MemoryStore
	useCase     *reopen_submission_usecase.UseCase
}

func newFixture(t *testing.T, verdict judgements.Verdict) fixture {
	t.Helper()
	f := fixture{
		submissions: submissions.NewMemoryStore(), judged: judgements.NewMemoryStore(), appeals: appeals.NewMemoryStore(),
	}
	row := submissions.TestWaiting(smurfs, ada, now.Add(-time.Hour))
	row.Name, row.State = "Greater Greece", submissions.Rejected
	row.Rejection = submissions.Rejection{Reason: submissions.Exists, Note: "Greece is in the game."}
	f.submissions.Seed(row)
	judgement, err := judgements.Snapshot{
		Submission: smurfs, Verdict: verdict, Why: "Greece's flag under another name", JudgedAt: now.Add(-time.Hour),
	}.Restore()
	require.NoError(t, err)
	require.NoError(t, f.judged.Save(t.Context(), judgement))
	f.useCase = reopen_submission_usecase.New(cptime.NewFixedClock(now), cpatomic.FakeTransactioner{},
		reopen_submission_usecase.Stores{Submissions: f.submissions, Judgements: f.judged, Appeals: f.appeals})
	return f
}

func TestTheOperatorReopensTheJudgesRejectionKeptAsItsFoundersAppeal(t *testing.T) {
	f := newFixture(t, judgements.Reject)

	out, err := f.useCase.Execute(t.Context(), reopen_submission_usecase.In{Submission: smurfs.String()})
	require.NoError(t, err)

	assert.Equal(t, submissions.Waiting, out.Submission.State)
	assert.Equal(t, judgements.Reject, out.Judgement.Verdict)
	assert.Equal(t, appeals.Snapshot{
		Submission: smurfs, Account: ada, State: appeals.Asked, AskedAt: now, DecidedAt: now,
		Rejection: submissions.Rejection{Reason: submissions.Exists, Note: "Greece is in the game."},
	}, out.Appeal)
	kept, err := f.submissions.Submission(t.Context(), smurfs)
	require.NoError(t, err)
	assert.Equal(t, submissions.Waiting, kept.TakeSnapshot().State)
	assert.Equal(t, appeals.Asked, f.appeals.Kept()[smurfs].State)
	assert.Equal(t, judgements.Reject, f.judged.Kept()[smurfs].Verdict, "never judged again: its judgement stays")

	_, err = f.useCase.Execute(t.Context(), reopen_submission_usecase.In{Submission: smurfs.String()})
	require.ErrorIs(t, err, appeals.ErrAskedAlready, "once per submission")
}

func TestTheOperatorReopensNeitherItsOwnRejectionNorAnUnknownSubmission(t *testing.T) {
	f := newFixture(t, judgements.Unsure)

	_, err := f.useCase.Execute(t.Context(), reopen_submission_usecase.In{Submission: smurfs.String()})
	require.ErrorIs(t, err, appeals.ErrNotTheJudges)
	_, err = f.useCase.Execute(t.Context(), reopen_submission_usecase.In{Submission: "nope"})
	require.ErrorIs(t, err, submissions.ErrUnknownSubmission)
	assert.Empty(t, f.appeals.Kept())
}
