package reject_submission_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions/usecases/reject_submission_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	id  = submissions.SubmissionID{15: 1}
)

func TestARejectionKeepsItsReasonForThePlayer(t *testing.T) {
	store := submissions.NewMemoryStore()
	store.Seed(submissions.TestWaiting(id, submissions.AccountID{15: 1}, now.Add(-time.Hour)))
	useCase := reject_submission_usecase.New(cptime.NewFixedClock(now), store)

	rejected, err := useCase.Execute(t.Context(), reject_submission_usecase.In{
		Submission: id.String(), Reason: submissions.Other, Note: "Rennes is a city: send Brittany's flag.",
	})
	require.NoError(t, err)

	assert.Equal(t, submissions.Rejected, rejected.State)
	assert.Equal(t, submissions.Rejection{Reason: submissions.Other, Note: "Rennes is a city: send Brittany's flag."},
		rejected.Rejection)
	assert.True(t, rejected.Held, "held 30 days, never public")
	assert.Equal(t, []submissions.Snapshot{rejected}, store.Kept())

	_, err = useCase.Execute(t.Context(), reject_submission_usecase.In{Submission: id.String(), Reason: submissions.Exists})
	require.ErrorIs(t, err, submissions.ErrDecided)
}

func TestOtherWithNoNoteIsNoRejection(t *testing.T) {
	store := submissions.NewMemoryStore()
	waiting := submissions.TestWaiting(id, submissions.AccountID{15: 1}, now)
	store.Seed(waiting)

	_, err := reject_submission_usecase.New(cptime.NewFixedClock(now), store).Execute(t.Context(),
		reject_submission_usecase.In{Submission: id.String(), Reason: submissions.Other})

	require.ErrorIs(t, err, submissions.ErrInvalidRejection)
	assert.Equal(t, []submissions.Snapshot{waiting}, store.Kept())
}
