package postgres_judgement_store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/judgements/postgres_judgement_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions/postgres_submission_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestTheWaitingSubmissionsWithNoJudgementAreJudgedTheFirstSentFirst(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "flags", migrations.FS)
	submitted, judged := postgres_submission_store.New(db), postgres_judgement_store.New(db)
	at := time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	for i, minutes := range []int{3, 1, 2, 0} {
		row := submissions.TestWaiting(submissions.SubmissionID{15: byte(i + 1)}, submissions.AccountID{15: byte(i + 1)},
			at.Add(time.Duration(minutes)*time.Minute))
		if i == 3 {
			row.State, row.Held = submissions.Approved, false
		}
		require.NoError(t, submitted.Save(t.Context(), submissions.Restored(row)))
	}
	unsure, err := judgements.Snapshot{
		Submission: submissions.SubmissionID{15: 2}, Verdict: judgements.Unsure, Why: "an unknown place", JudgedAt: at,
	}.Restore()
	require.NoError(t, err)
	require.NoError(t, judged.Save(t.Context(), unsure))
	require.NoError(t, judged.Save(t.Context(), unsure), "kept once")

	unjudged, err := judged.Unjudged(t.Context(), 5)
	require.NoError(t, err)
	assert.Equal(t, []submissions.SubmissionID{{15: 3}, {15: 1}}, unjudged, "not the one judged, nor the one decided")

	first, err := judged.Unjudged(t.Context(), 1)
	require.NoError(t, err)
	assert.Equal(t, []submissions.SubmissionID{{15: 3}}, first)

	require.NoError(t, submitted.DeleteWaiting(t.Context(), submissions.SubmissionID{15: 2}),
		"a submission replaced takes its judgement with it")
}
