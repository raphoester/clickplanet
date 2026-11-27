package post_review_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/reviews"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/reviews/usecases/post_review_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 11, 27, 9, 0, 0, 0, time.UTC)

func setUp() (*pings.RecordedPoster, *post_review_usecase.UseCase) {
	poster := &pings.RecordedPoster{}
	return poster, post_review_usecase.New(cptime.NewFixedClock(now), reviews.NewInMemoryStore(),
		reviews.NewDesk(poster, "300", "https://backoffice.clickplanet.lol"), cpatomic.FakeTransactioner{})
}

func review(cause reviews.Cause) reviews.Review {
	return reviews.Review{
		Submission: "00000000-0000-0000-0000-000000000001", Name: "Solandia", Cause: cause, Why: "could be fictional",
		AskedAt: now,
	}
}

func TestEachCauseOfASubmissionIsToldOnce(t *testing.T) {
	poster, useCase := setUp()

	require.NoError(t, useCase.Execute(t.Context(), review(reviews.Unsure)))
	require.NoError(t, useCase.Execute(t.Context(), review(reviews.Unsure)))
	require.NoError(t, useCase.Execute(t.Context(), review(reviews.Appeal)))

	assert.Len(t, poster.Posts, 2)
}

func TestAPostDiscordRefusedIsTriedAgain(t *testing.T) {
	poster, useCase := setUp()
	poster.Err = discord.ErrRefused

	require.ErrorIs(t, useCase.Execute(t.Context(), review(reviews.Appeal)), discord.ErrRefused)

	poster.Err = nil
	require.NoError(t, useCase.Execute(t.Context(), review(reviews.Appeal)))
	assert.Len(t, poster.Posts, 1)
}
