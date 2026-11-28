package reopen_submission_handler_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/appeals"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/appeals/usecases/reopen_submission_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/flagmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/reopen_submission_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/submissionmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
)

var at = time.Date(2026, 11, 28, 9, 0, 0, 0, time.UTC)

type stubUseCase struct {
	out reopen_submission_usecase.Out
	err error
}

func (s stubUseCase) Execute(context.Context, reopen_submission_usecase.In) (reopen_submission_usecase.Out, error) {
	return s.out, s.err
}

type noLinks struct{}

func (noLinks) HeldURL(string, time.Duration) (string, error) { return "https://held.test", nil }

// The submission answered, or the code of the refusal.
func reopen(useCase stubUseCase) (*flagsv1.KeptSubmission, connect.Code) {
	answer, err := reopen_submission_handler.New(useCase, submissionmessage.New(flagmessage.New("https://media.test"), noLinks{})).
		ReopenSubmission(context.Background(), connect.NewRequest(&flagsv1.ReopenSubmissionRequest{SubmissionId: "s"}))
	if err != nil {
		return nil, connect.CodeOf(err)
	}
	return answer.Msg.GetSubmission(), 0
}

func TestTheReopenedSubmissionWaitsWithTheJudgesRejectAndItsAppeal(t *testing.T) {
	id := submissions.SubmissionID{15: 1}
	rejected := submissions.Rejection{Reason: submissions.Exists, Note: "Greece is in the game."}
	kept, code := reopen(stubUseCase{out: reopen_submission_usecase.Out{
		Submission: submissions.TestWaiting(id, catalog.AccountID{15: 1}, at),
		Judgement:  judgements.Snapshot{Submission: id, Verdict: judgements.Reject, Why: "Greece's flag", JudgedAt: at},
		Appeal: appeals.Snapshot{
			Submission: id, Account: catalog.AccountID{15: 1}, Rejection: rejected, State: appeals.Asked, AskedAt: at,
			DecidedAt: at,
		},
	}})
	require.Zero(t, code)

	assert.Equal(t, flagsv1.SubmissionState_SUBMISSION_STATE_WAITING, kept.GetState())
	assert.Equal(t, flagsv1.JudgementVerdict_JUDGEMENT_VERDICT_REJECT, kept.GetJudgement().GetVerdict())
	assert.Equal(t, flagsv1.RejectionReason_REJECTION_REASON_EXISTS, kept.GetAppeal().GetRejection().GetReason())
	assert.Equal(t, "https://held.test", kept.GetPicture().GetUrl(), "still held in private")
}

func TestARefusedReopenIsNotFoundOrAFailedPrecondition(t *testing.T) {
	for err, code := range map[error]connect.Code{
		submissions.ErrUnknownSubmission: connect.CodeNotFound,
		submissions.ErrNotRejected:       connect.CodeFailedPrecondition,
		submissions.ErrPictureGone:       connect.CodeFailedPrecondition,
		submissions.ErrWaiting:           connect.CodeFailedPrecondition,
		appeals.ErrNotTheJudges:          connect.CodeFailedPrecondition,
		appeals.ErrAskedAlready:          connect.CodeFailedPrecondition,
	} {
		_, refused := reopen(stubUseCase{err: err})
		assert.Equal(t, code, refused, err.Error())
	}
}
