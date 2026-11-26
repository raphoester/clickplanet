package submissions_query_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/postgres_flag_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/flagmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/submissionmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/submissions_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/grants"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/grants/postgres_grant_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/judgements/postgres_judgement_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions/postgres_submission_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

type links struct{}

func (links) HeldURL(key string, _ time.Duration) (string, error) {
	return "https://bucket.test/" + key, nil
}

var (
	at  = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	ada = catalog.AccountID{15: 1}
	bob = catalog.AccountID{15: 2}
)

func ids[T interface{ GetId() string }](messages []T) []string {
	read := make([]string, 0, len(messages))
	for _, message := range messages {
		read = append(read, message.GetId()[34:])
	}
	return read
}

func TestTheOperatorReadsTheWaitingOldestFirstThenTheDecidedLatestFirst(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "flags", migrations.FS)
	store := postgres_submission_store.New(db)
	query := submissions_query.NewPostgresQuery(db,
		submissionmessage.New(flagmessage.New("https://media.test"), links{}), submissions_query.EveryWaiting)

	late := submissions.TestWaiting(submissions.SubmissionID{15: 1}, ada, at.Add(2*time.Hour))
	early := submissions.TestWaiting(submissions.SubmissionID{15: 2}, bob, at)
	rejected := submissions.TestWaiting(submissions.SubmissionID{15: 3}, ada, at.Add(-48*time.Hour))
	rejected.State, rejected.DecidedAt = submissions.Rejected, at.Add(-24*time.Hour)
	rejected.Rejection = submissions.Rejection{Reason: submissions.Other, Note: "A city."}
	approved := submissions.TestWaiting(submissions.SubmissionID{15: 4}, bob, at.Add(-72*time.Hour))
	approved.State, approved.DecidedAt, approved.Held = submissions.Approved, at.Add(-time.Hour), false
	for _, row := range []submissions.Snapshot{late, early, rejected, approved} {
		require.NoError(t, store.Save(t.Context(), submissions.Restored(row)))
	}

	kept, err := query.Kept(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []string{"02", "01", "04", "03"}, ids(kept.GetSubmissions()))
	assert.Equal(t, "https://bucket.test/"+early.ID.PictureKey(), kept.GetSubmissions()[0].GetPicture().GetUrl())
	assert.Equal(t, "A city.", kept.GetSubmissions()[3].GetRejection().GetNote())

	mine, err := query.Mine(t.Context(), ada)
	require.NoError(t, err)
	assert.Equal(t, []string{"01", "03"}, ids(mine.GetSubmissions()), "the newest first")
	assert.Equal(t, flagsv1.SubmissionState_SUBMISSION_STATE_REJECTED, mine.GetSubmissions()[1].GetState())
}

func TestAHolderReadsItsFlagsAndWhyTheOperatorRemovedThemAndNoOtherFlag(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "flags", migrations.FS)
	flags := postgres_flag_store.New(db)
	query := submissions_query.NewPostgresQuery(db,
		submissionmessage.New(flagmessage.New("https://media.test"), links{}), submissions_query.EveryWaiting)

	held := func(flag catalog.FlagID, name catalog.Name, account catalog.AccountID, note catalog.Note) catalog.Flag {
		t.Helper()
		opened := catalog.HeldFlag(flag, name, account, at)
		if note != "" {
			removed, err := opened.Removed(note, at.Add(time.Hour))
			require.NoError(t, err)
			opened = removed
		}
		require.NoError(t, flags.Save(t.Context(), opened))
		return opened
	}
	held("r-0000000001", "Bavaria", ada, "Broke the rules.")
	held("r-0000000002", "Swabia", ada, "")
	held("r-0000000003", "Tyrol", bob, "A test.")
	claimed, ok := held("r-0000000004", "Savoy", ada, "").Expired(at.AddDate(0, 2, 0))
	require.True(t, ok)
	claimed, _, err := claimed.Paid(bob, at.AddDate(0, 2, 1), grants.Week, at.AddDate(0, 2, 1))
	require.NoError(t, err)
	require.NoError(t, flags.Save(t.Context(), claimed))

	mine, err := query.Mine(t.Context(), ada)
	require.NoError(t, err)
	assert.Equal(t, []string{"r-0000000001", "r-0000000002"}, mine.GetHeld(), "Savoy left Ada's flags once Bob claimed it")
	require.Len(t, mine.GetClosures(), 1, "not a flag open, nor one another account holds")
	assert.Equal(t, "r-0000000001", mine.GetClosures()[0].GetFlag())
	assert.Equal(t, "Broke the rules.", mine.GetClosures()[0].GetNote())
}

func TestWhileTheShopSellsTheOperatorReadsOnlyTheFlagsPaidFor(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "flags", migrations.FS)
	store, paidTime := postgres_submission_store.New(db), postgres_grant_store.New(db)
	query := submissions_query.NewPostgresQuery(db,
		submissionmessage.New(flagmessage.New("https://media.test"), links{}), submissions_query.PaidWaiting)

	for i := byte(1); i <= 3; i++ {
		row := submissions.TestWaiting(submissions.SubmissionID{15: i}, ada, at.Add(time.Duration(i)*time.Hour))
		require.NoError(t, store.Save(t.Context(), submissions.Restored(row)))
	}
	hold := func(id grants.GrantID, submission byte, takenBack bool) {
		t.Helper()
		held := grants.Snapshot{
			ID: id, Held: true, Submission: submissions.SubmissionID{15: submission}, Period: grants.Month, HeldAt: at,
			TakenBack: takenBack, TakenBackAt: at,
		}
		grant, err := held.Restore()
		require.NoError(t, err)
		require.NoError(t, db.InTx(t.Context(), func(ctx context.Context) error { return paidTime.Save(ctx, grant) }))
	}
	hold("stripe:in_1", 1, false)
	hold("stripe:in_2", 2, true)

	kept, err := query.Kept(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []string{"01"}, ids(kept.GetSubmissions()), "the one refunded and the one never paid wait unseen")

	mine, err := query.Mine(t.Context(), ada)
	require.NoError(t, err)
	paid := map[string]bool{}
	for _, submission := range mine.GetSubmissions() {
		paid[submission.GetId()[34:]] = submission.GetPaid()
	}
	assert.Equal(t, map[string]bool{"01": true, "02": false, "03": false}, paid)

	submission, err := store.Submission(t.Context(), submissions.SubmissionID{15: 1})
	require.NoError(t, err)
	assert.True(t, submission.TakeSnapshot().Paid)
}

func TestTheOperatorReadsWhatTheJudgeRuled(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "flags", migrations.FS)
	store := postgres_submission_store.New(db)
	query := submissions_query.NewPostgresQuery(db,
		submissionmessage.New(flagmessage.New("https://media.test"), links{}), submissions_query.EveryWaiting)
	for i := byte(1); i <= 2; i++ {
		row := submissions.TestWaiting(submissions.SubmissionID{15: i}, ada, at.Add(time.Duration(i)*time.Hour))
		require.NoError(t, store.Save(t.Context(), submissions.Restored(row)))
	}
	unsure, err := judgements.Snapshot{
		Submission: submissions.SubmissionID{15: 1}, Verdict: judgements.Unsure, Why: "an unknown place", JudgedAt: at,
	}.Restore()
	require.NoError(t, err)
	require.NoError(t, postgres_judgement_store.New(db).Save(t.Context(), unsure))

	kept, err := query.Kept(t.Context())
	require.NoError(t, err)

	require.Len(t, kept.GetSubmissions(), 2)
	judged := kept.GetSubmissions()[0].GetJudgement()
	assert.Equal(t, flagsv1.JudgementVerdict_JUDGEMENT_VERDICT_UNSURE, judged.GetVerdict())
	assert.Equal(t, "an unknown place", judged.GetWhy())
	assert.Nil(t, kept.GetSubmissions()[1].GetJudgement(), "never judged")
}
