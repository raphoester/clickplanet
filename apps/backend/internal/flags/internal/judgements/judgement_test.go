package judgements_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
)

var (
	at       = time.Date(2026, 11, 27, 9, 0, 0, 0, time.UTC)
	solandia = submissions.SubmissionID{15: 1}
)

func judged(t *testing.T, verdict judgements.Verdict) judgements.Judgement {
	t.Helper()
	judgement, err := judgements.Snapshot{
		Submission: solandia, Verdict: verdict, Why: "could be fictional", JudgedAt: at,
	}.Restore()
	require.NoError(t, err)
	return judgement
}

func waiting() submissions.Submission {
	return submissions.Restored(submissions.TestWaiting(solandia, catalog.AccountID{15: 1}, at))
}

func TestAnUnsureJudgementOfOneWaitingPaidForAsksTheOperator(t *testing.T) {
	facts := judged(t, judgements.Unsure).Facts(waiting().PaidFor(), at.Add(time.Minute))

	require.Len(t, facts, 1)
	assert.True(t, proto.Equal(&flagsv1.ReviewAsked{
		SubmissionId: solandia.String(), Name: "Rennes", Cause: flagsv1.ReviewCause_REVIEW_CAUSE_UNSURE,
		Why: "could be fictional", AskedAt: timestamppb.New(at.Add(time.Minute)),
	}, facts[0]))
}

func TestNoOtherJudgementOrSubmissionAsksTheOperator(t *testing.T) {
	assert.Empty(t, judged(t, judgements.Unsure).Facts(waiting(), at), "not paid for: the shop sells it first")
	assert.Empty(t, judged(t, judgements.Approve).Facts(waiting().PaidFor(), at))
	assert.Empty(t, judged(t, judgements.Reject).Facts(waiting().PaidFor(), at))

	rejected, err := waiting().PaidFor().Rejected(submissions.Rejection{Reason: submissions.Exists}, at)
	require.NoError(t, err)
	assert.Empty(t, judged(t, judgements.Unsure).Facts(rejected, at), "decided already")
}

func TestOnlyARejectionTheJudgeMadeIsRejected(t *testing.T) {
	assert.True(t, judged(t, judgements.Reject).Rejected())
	assert.False(t, judged(t, judgements.Unsure).Rejected())
	assert.False(t, judged(t, judgements.Approve).Rejected())
	assert.False(t, judgements.Judgement{}.Rejected(), "none")
}
