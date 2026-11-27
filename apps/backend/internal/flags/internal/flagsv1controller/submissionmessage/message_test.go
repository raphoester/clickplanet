package submissionmessage_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/appeals"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/flagmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/submissionmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
)

type links struct{}

func (links) HeldURL(key string, ttl time.Duration) (string, error) {
	return "https://bucket.test/" + key + "?for=" + ttl.String(), nil
}

var (
	at       = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	messages = submissionmessage.New(flagmessage.New("https://media.test/"), links{})
	waiting  = submissions.TestWaiting(submissions.SubmissionID{15: 1}, catalog.AccountID{15: 1}, at)
	public   = "https://media.test/flags/pictures/" + waiting.Picture.Hash + ".png"
)

func TestAPlayerSeesItsPictureOnlyOnceItIsPublic(t *testing.T) {
	assert.Nil(t, messages.Submission(waiting, submissionmessage.Reviews{}).GetPicture())

	approved := waiting
	approved.State, approved.Held = submissions.Approved, false
	assert.Equal(t, public, messages.Submission(approved, submissionmessage.Reviews{}).GetPicture().GetUrl())

	rejected := waiting
	rejected.State, rejected.Rejection = submissions.Rejected, submissions.Rejection{Reason: submissions.Other, Note: "A city."}
	message := messages.Submission(rejected, submissionmessage.Reviews{})
	assert.Nil(t, message.GetPicture())
	assert.Equal(t, flagsv1.RejectionReason_REJECTION_REASON_OTHER, message.GetRejection().GetReason())
	assert.Equal(t, "A city.", message.GetRejection().GetNote())
}

func TestTheOperatorSeesAHeldPictureThroughALinkOfAnHour(t *testing.T) {
	kept, err := messages.Kept(waiting)
	require.NoError(t, err)
	assert.Equal(t, "https://bucket.test/"+waiting.ID.PictureKey()+"?for=1h0m0s", kept.GetPicture().GetUrl())
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", kept.GetAccountId())
	assert.Equal(t, uint32(3), kept.GetOtherColorsPercent())

	deleted := waiting
	deleted.State, deleted.Held = submissions.Rejected, false
	deleted.Rejection = submissions.Rejection{Reason: submissions.Hateful}
	kept, err = messages.Kept(deleted)
	require.NoError(t, err)
	assert.Nil(t, kept.GetPicture(), "a rejected picture deleted after 30 days")
}

func TestEveryReasonCrossesTheWireBothWays(t *testing.T) {
	for _, reason := range []submissions.Reason{
		submissions.NotAPlace, submissions.Exists, submissions.Unreadable, submissions.Hateful, submissions.Other,
	} {
		assert.Equal(t, reason, submissionmessage.ReasonOf(submissionmessage.Reason(reason)))
	}
	assert.Equal(t, submissions.Reason(""), submissionmessage.ReasonOf(flagsv1.RejectionReason_REJECTION_REASON_UNSPECIFIED))
}

func TestAPlayerReadsWhetherTheJudgeRejectedItAndItsAppeal(t *testing.T) {
	rejected := waiting
	rejected.State, rejected.Rejection = submissions.Rejected, submissions.Rejection{Reason: submissions.NotAPlace}
	byJudge := judgements.Snapshot{Submission: waiting.ID, Verdict: judgements.Reject, Why: "a club", JudgedAt: at}
	assert.True(t, messages.Submission(rejected, submissionmessage.Reviews{Judgement: byJudge}).GetRejectedByJudge())

	unsure := byJudge
	unsure.Verdict = judgements.Unsure
	assert.False(t, messages.Submission(rejected, submissionmessage.Reviews{Judgement: unsure}).GetRejectedByJudge(),
		"the operator's")

	appeal := appeals.Snapshot{
		Submission: waiting.ID, Account: waiting.Account, Rejection: rejected.Rejection, State: appeals.Answered,
		AskedAt: at.Add(time.Hour), DecidedAt: at.Add(2 * time.Hour),
	}
	final := messages.Submission(rejected, submissionmessage.Reviews{Judgement: byJudge, Appeal: appeal})
	assert.False(t, final.GetRejectedByJudge(), "rejected again by the operator: final")
	assert.Equal(t, at.Add(time.Hour), final.GetAppeal().GetAskedAt().AsTime())
	assert.Equal(t, at.Add(2*time.Hour), final.GetAppeal().GetAnsweredAt().AsTime())
	assert.Equal(t, flagsv1.RejectionReason_REJECTION_REASON_NOT_A_PLACE, final.GetAppeal().GetRejection().GetReason())

	appeal.State = appeals.Asked
	asked := messages.Submission(waiting, submissionmessage.Reviews{Judgement: byJudge, Appeal: appeal})
	assert.Nil(t, asked.GetAppeal().GetAnsweredAt(), "it waits for the operator")
	assert.Nil(t, messages.Submission(waiting, submissionmessage.Reviews{}).GetAppeal())
}
