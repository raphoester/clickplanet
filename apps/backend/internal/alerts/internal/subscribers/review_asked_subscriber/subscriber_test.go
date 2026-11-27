package review_asked_subscriber_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/reviews"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers/review_asked_subscriber"
)

type useCase struct{ heard []reviews.Review }

func (u *useCase) Execute(_ context.Context, review reviews.Review) error {
	u.heard = append(u.heard, review)
	return nil
}

func TestTheFlagsReviewAskedIsToldToTheStaff(t *testing.T) {
	at := time.Date(2026, 11, 27, 9, 0, 0, 0, time.UTC)
	heard := &useCase{}
	subscriber := review_asked_subscriber.New(heard)

	require.NoError(t, subscriber.Handle(t.Context(), &flagsv1.ReviewAsked{
		SubmissionId: "00000000-0000-0000-0000-000000000001", Name: "Smurfland",
		Cause: flagsv1.ReviewCause_REVIEW_CAUSE_APPEAL, Why: "a cartoon's village", AskedAt: timestamppb.New(at),
	}))
	require.NoError(t, subscriber.Handle(t.Context(), &flagsv1.ReviewAsked{
		SubmissionId: "00000000-0000-0000-0000-000000000002", Name: "Solandia",
		Cause: flagsv1.ReviewCause_REVIEW_CAUSE_UNSPECIFIED,
	}), "a cause this build does not know")

	assert.Equal(t, []reviews.Review{{
		Submission: "00000000-0000-0000-0000-000000000001", Name: "Smurfland", Cause: reviews.Appeal,
		Why: "a cartoon's village", AskedAt: at,
	}}, heard.heard)
}
