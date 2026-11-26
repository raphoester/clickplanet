package reject_submission_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions/usecases/reject_submission_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	id  = submissions.SubmissionID{15: 1}
)

func TestARejectionKeepsItsReasonForThePlayer(t *testing.T) {
	store := submissions.NewMemoryStore()
	store.Seed(submissions.TestWaiting(id, catalog.AccountID{15: 1}, now.Add(-time.Hour)))
	outbox := &cpatomic.FakeOutbox{}
	useCase := reject_submission_usecase.New(cptime.NewFixedClock(now), cpatomic.FakeTransactioner{}, store, outbox)

	rejected, err := useCase.Execute(t.Context(), reject_submission_usecase.In{
		Submission: id.String(), Reason: submissions.Other, Note: "Rennes is a city: send Brittany's flag.",
	})
	require.NoError(t, err)

	assert.Equal(t, submissions.Rejected, rejected.State)
	assert.Equal(t, submissions.Rejection{Reason: submissions.Other, Note: "Rennes is a city: send Brittany's flag."},
		rejected.Rejection)
	assert.True(t, rejected.Held, "held 30 days, never public")
	assert.Equal(t, []submissions.Snapshot{rejected}, store.Kept())
	require.Len(t, outbox.Recorded(), 1, "the shop refunds what was paid for it")
	assert.True(t, proto.Equal(&flagsv1.SubmissionRejected{SubmissionId: id.String(), RejectedAt: timestamppb.New(now)},
		outbox.Recorded()[0]))

	_, err = useCase.Execute(t.Context(), reject_submission_usecase.In{Submission: id.String(), Reason: submissions.Exists})
	require.ErrorIs(t, err, submissions.ErrDecided)
}

func TestOtherWithNoNoteIsNoRejection(t *testing.T) {
	store := submissions.NewMemoryStore()
	waiting := submissions.TestWaiting(id, catalog.AccountID{15: 1}, now)
	store.Seed(waiting)

	_, err := reject_submission_usecase.New(cptime.NewFixedClock(now), cpatomic.FakeTransactioner{}, store,
		&cpatomic.FakeOutbox{}).Execute(t.Context(),
		reject_submission_usecase.In{Submission: id.String(), Reason: submissions.Other})

	require.ErrorIs(t, err, submissions.ErrInvalidRejection)
	assert.Equal(t, []submissions.Snapshot{waiting}, store.Kept())
}
