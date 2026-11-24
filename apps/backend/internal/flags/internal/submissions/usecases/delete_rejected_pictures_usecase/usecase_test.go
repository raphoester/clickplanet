package delete_rejected_pictures_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions/usecases/delete_rejected_pictures_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 12, 24, 9, 0, 0, 0, time.UTC)

func rejected(id submissions.SubmissionID, at time.Time) submissions.Snapshot {
	submission := submissions.TestWaiting(id, submissions.AccountID{15: 1}, at)
	submission.State, submission.DecidedAt = submissions.Rejected, at
	submission.Rejection = submissions.Rejection{Reason: submissions.NotAPlace}
	return submission
}

func TestARejectedPictureIsDeleted30DaysAfterItsRejection(t *testing.T) {
	old, recent := submissions.SubmissionID{15: 1}, submissions.SubmissionID{15: 2}
	store, bucket := submissions.NewMemoryStore(), pictures.NewMemoryBucket()
	store.Seed(rejected(old, now.Add(-30*24*time.Hour)), rejected(recent, now.Add(-29*24*time.Hour)))
	for _, id := range []submissions.SubmissionID{old, recent} {
		require.NoError(t, bucket.Hold(t.Context(), id.PictureKey(), []byte("png")))
	}

	require.NoError(t, delete_rejected_pictures_usecase.New(cptime.NewFixedClock(now), store,
		pictures.NewGallery(bucket, pictures.Rules{MaxBytes: 5 << 20})).Execute(t.Context()))

	assert.Equal(t, map[string][]byte{recent.PictureKey(): []byte("png")}, bucket.HeldPictures())
	held := map[submissions.SubmissionID]bool{}
	for _, kept := range store.Kept() {
		held[kept.ID] = kept.Held
	}
	assert.Equal(t, map[submissions.SubmissionID]bool{old: false, recent: true}, held)
}
