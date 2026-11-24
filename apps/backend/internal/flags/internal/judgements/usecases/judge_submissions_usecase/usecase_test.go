package judge_submissions_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/judgements/review_verdicts"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/judgements/usecases/judge_submissions_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions/usecases/approve_submission_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions/usecases/reject_submission_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)

// judge rules on each name as told, and counts what it was asked.
type judge struct {
	rulings map[catalog.Name]judgements.Ruling
	fails   error
	asked   []judgements.Case
}

func (j *judge) Judge(_ context.Context, sent judgements.Case) (judgements.Ruling, error) {
	j.asked = append(j.asked, sent)
	if j.fails != nil {
		return judgements.Ruling{}, j.fails
	}
	return j.rulings[sent.Name], nil
}

type fixture struct {
	submissions *submissions.MemoryStore
	flags       *catalog.MemoryStore
	judged      *judgements.MemoryStore
	outbox      *cpatomic.FakeOutbox
	judge       *judge
	useCase     *judge_submissions_usecase.UseCase
}

func newFixture(t *testing.T, names ...catalog.Name) fixture {
	t.Helper()
	f := fixture{
		submissions: submissions.NewMemoryStore(), flags: catalog.NewMemoryStore(), outbox: &cpatomic.FakeOutbox{},
		judge: &judge{rulings: map[catalog.Name]judgements.Ruling{}},
	}
	gallery := pictures.NewGallery(pictures.NewMemoryBucket(), pictures.Rules{MaxBytes: 5 << 20})
	drawn, err := pictures.Redraw(pictures.TestPNG(300, 200))
	require.NoError(t, err)
	ids := make([]submissions.SubmissionID, 0, len(names))
	for i, name := range names {
		id := submissions.SubmissionID{15: byte(i + 1)}
		require.NoError(t, gallery.Hold(t.Context(), id.PictureKey(), drawn))
		row := submissions.TestWaiting(id, submissions.AccountID{15: byte(i + 1)}, now.Add(time.Duration(i)*time.Minute))
		row.Name, row.Picture = name, drawn.Picture.TakeSnapshot()
		f.submissions.Seed(row)
		ids = append(ids, id)
	}
	f.judged = judgements.NewMemoryStore(ids...)
	clock := cptime.NewFixedClock(now)
	verdicts := review_verdicts.New(
		approve_submission_usecase.New(clock, gallery, cpatomic.FakeTransactioner{},
			approve_submission_usecase.Stores{Submissions: f.submissions, Flags: f.flags}, f.outbox),
		reject_submission_usecase.New(clock, cpatomic.FakeTransactioner{}, f.submissions, f.outbox),
	)
	f.useCase = judge_submissions_usecase.New(clock, f.judged,
		judgements.Cases{Submissions: f.submissions, Pictures: gallery, Flags: f.flags}, f.judge, verdicts)
	return f
}

func (f fixture) state(t *testing.T, id submissions.SubmissionID) submissions.Snapshot {
	t.Helper()
	submission, err := f.submissions.Submission(t.Context(), id)
	require.NoError(t, err)
	return submission.TakeSnapshot()
}

func TestTheJudgeApprovesRejectsOrLeavesAFlagToTheOperator(t *testing.T) {
	f := newFixture(t, "Bavaria", "Rainbow", "Ruritania")
	pride := submissions.Rejection{Reason: submissions.NotAPlace, Note: "A pride flag is no place's flag."}
	f.judge.rulings["Bavaria"] = judgements.Ruling{Verdict: judgements.Approve, Why: "Bavaria's lozenges"}
	f.judge.rulings["Rainbow"] = judgements.Ruling{Verdict: judgements.Reject, Rejection: pride, Why: "a pride flag"}
	f.judge.rulings["Ruritania"] = judgements.Ruling{Verdict: judgements.Unsure, Why: "an unknown place"}

	judged, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 3, judged)

	assert.Equal(t, submissions.Approved, f.state(t, submissions.SubmissionID{15: 1}).State)
	rejected := f.state(t, submissions.SubmissionID{15: 2})
	assert.Equal(t, submissions.Rejected, rejected.State)
	assert.Equal(t, pride, rejected.Rejection)
	assert.Equal(t, submissions.Waiting, f.state(t, submissions.SubmissionID{15: 3}).State, "the operator decides")
	require.Len(t, f.outbox.Recorded(), 2)
	assert.IsType(t, &flagsv1.SubmissionApproved{}, f.outbox.Recorded()[0], "a flag paid for opens")
	assert.IsType(t, &flagsv1.SubmissionRejected{}, f.outbox.Recorded()[1], "a flag paid for is refunded")
	assert.Equal(t, judgements.Unsure, f.judged.Kept()[submissions.SubmissionID{15: 3}].Verdict)
	assert.Len(t, f.judge.asked, 3)
	assert.Equal(t, []catalog.Name{}, f.judge.asked[0].Taken)
	assert.NotEmpty(t, f.judge.asked[0].Picture, "the picture held in private")

	again, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)
	assert.Zero(t, again)
	assert.Len(t, f.judge.asked, 3, "each judged once, the unsure one too")
}

func TestAJudgeThatCannotBeAskedIsAskedAgainNextRound(t *testing.T) {
	f := newFixture(t, "Bavaria")
	f.judge.fails = errors.New("overloaded")

	_, err := f.useCase.Execute(t.Context())
	require.Error(t, err)
	assert.Empty(t, f.judged.Kept())

	f.judge.fails = nil
	f.judge.rulings["Bavaria"] = judgements.Ruling{Verdict: judgements.Approve, Why: "Bavaria's lozenges"}
	judged, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, judged)
}

func TestAnApprovalOfANameTakenSinceIsLeftToTheOperator(t *testing.T) {
	f := newFixture(t, "Bavaria")
	require.NoError(t, f.flags.Save(t.Context(), catalog.OpenFlag("r-0000000001", "Bavaria", now.Add(-time.Hour))))
	f.judge.rulings["Bavaria"] = judgements.Ruling{Verdict: judgements.Approve, Why: "Bavaria's lozenges"}

	_, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)

	assert.Equal(t, submissions.Waiting, f.state(t, submissions.SubmissionID{15: 1}).State)
	kept := f.judged.Kept()[submissions.SubmissionID{15: 1}]
	assert.Equal(t, judgements.Unsure, kept.Verdict)
	assert.Equal(t, []catalog.Name{"Bavaria"}, f.judge.asked[0].Taken)
}

func TestAFlagTheOperatorDecidedFirstIsNotJudged(t *testing.T) {
	f := newFixture(t, "Bavaria")
	decided := submissions.TestWaiting(submissions.SubmissionID{15: 1}, submissions.AccountID{15: 1}, now)
	decided.State, decided.Held = submissions.Approved, false
	f.submissions.Seed(decided)

	judged, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)

	assert.Zero(t, judged)
	assert.Empty(t, f.judge.asked)
}
