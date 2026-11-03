package announce_ranking_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings/usecases/announce_ranking_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/roles"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 11, 3, 21, 0, 5, 0, time.UTC)

type fakeResults struct {
	ranking pings.Ranking
	err     error
	asked   int
}

func (f *fakeResults) RankingAfter(_ context.Context, _ pings.RoundID) (pings.Ranking, error) {
	f.asked++
	return f.ranking, f.err
}

var scored = pings.Ranking{
	Won:   []pings.Place{{Flag: "ie", Rank: 1, Points: 25}},
	Table: []pings.Place{{Flag: "ie", Rank: 1, Points: 25}},
}

func setUp(results *fakeResults) (*pings.RecordedPoster, *pings.InMemoryStore, *announce_ranking_usecase.UseCase) {
	poster := &pings.RecordedPoster{}
	store := pings.NewInMemoryStore()
	herald := pings.NewHerald(roles.NewBook(roles.NewInMemoryStore(), &roles.FakeMaker{}), poster,
		pings.Channels{Alerts: "100", Battles: "200"}, "https://clickplanet.lol/play")
	return poster, store, announce_ranking_usecase.New(
		cptime.NewFixedClock(now), store, results, herald, cpatomic.FakeTransactioner{})
}

func day(endedAt time.Time) pings.EndedDay {
	return pings.EndedDay{Round: "season-1/day-3", Season: 1, Number: 3, EndedAt: endedAt}
}

func TestADaysRankingIsToldOnceInAlerts(t *testing.T) {
	results := &fakeResults{ranking: scored}
	poster, _, useCase := setUp(results)

	require.NoError(t, useCase.Execute(t.Context(), day(now.Add(-5*time.Second))))
	require.NoError(t, useCase.Execute(t.Context(), day(now.Add(-5*time.Second))))

	require.Len(t, poster.Posts, 1)
	assert.Equal(t, discord.ChannelID("100"), poster.Posts[0].Channel)
	assert.Equal(t, 1, results.asked)
}

func TestARankingToldLateIsNotTold(t *testing.T) {
	results := &fakeResults{ranking: scored}
	poster, _, useCase := setUp(results)

	require.NoError(t, useCase.Execute(t.Context(), day(now.Add(-pings.RankingFresh-time.Second))))

	assert.Empty(t, poster.Posts)
	assert.Zero(t, results.asked)
}

func TestADayNobodyScoredInTellsNothing(t *testing.T) {
	poster, _, useCase := setUp(&fakeResults{})

	require.NoError(t, useCase.Execute(t.Context(), day(now)))

	assert.Empty(t, poster.Posts)
}

func TestARoundScoringCouldNotReadIsTriedAgain(t *testing.T) {
	unread := errors.New("scoring is down")
	results := &fakeResults{err: unread}
	poster, store, useCase := setUp(results)

	require.ErrorIs(t, useCase.Execute(t.Context(), day(now)), unread)
	pinged, err := store.RankingPinged(t.Context(), "season-1/day-3")
	require.NoError(t, err)
	require.False(t, pinged)

	results.err, results.ranking = nil, scored
	require.NoError(t, useCase.Execute(t.Context(), day(now)))
	assert.Len(t, poster.Posts, 1)
}

func TestAPostDiscordRefusedIsTriedAgain(t *testing.T) {
	poster, store, useCase := setUp(&fakeResults{ranking: scored})
	poster.Err = discord.ErrRefused

	require.ErrorIs(t, useCase.Execute(t.Context(), day(now)), discord.ErrRefused)
	pinged, err := store.RankingPinged(t.Context(), "season-1/day-3")
	require.NoError(t, err)
	require.False(t, pinged)

	poster.Err = nil
	require.NoError(t, useCase.Execute(t.Context(), day(now)))
	assert.Len(t, poster.Posts, 1)
}
