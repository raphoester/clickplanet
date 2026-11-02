package announce_start_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings/usecases/announce_start_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/roles"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 11, 2, 18, 0, 0, 0, time.UTC)

func setUp() (*pings.RecordedPoster, *pings.InMemoryStore, *announce_start_usecase.UseCase) {
	poster := &pings.RecordedPoster{}
	store := pings.NewInMemoryStore()
	herald := pings.NewHerald(roles.NewBook(roles.NewInMemoryStore(), &roles.FakeMaker{}), poster,
		pings.Channels{Alerts: "100", Battles: "200"}, "https://clickplanet.lol/play")
	return poster, store, announce_start_usecase.New(cptime.NewFixedClock(now), store, herald, cpatomic.FakeTransactioner{})
}

func battle(startsAt time.Time) pings.Start {
	return pings.Start{Key: "season-1/battle-1", Kind: pings.BattleStart, Times: 2, StartsAt: startsAt, EndsAt: startsAt.Add(time.Hour)}
}

func TestABattleIsToldOnce(t *testing.T) {
	poster, _, useCase := setUp()

	require.NoError(t, useCase.Execute(t.Context(), battle(now.Add(-time.Minute))))
	require.NoError(t, useCase.Execute(t.Context(), battle(now.Add(-time.Minute))))

	require.Len(t, poster.Posts, 1)
	assert.Equal(t, discord.ChannelID("200"), poster.Posts[0].Channel)
}

func TestAStartToldLateIsNotTold(t *testing.T) {
	poster, _, useCase := setUp()

	require.NoError(t, useCase.Execute(t.Context(), battle(now.Add(-pings.Fresh-time.Second))))

	assert.Empty(t, poster.Posts)
}

func TestAPostDiscordRefusedIsTriedAgain(t *testing.T) {
	poster, store, useCase := setUp()
	poster.Err = discord.ErrRefused

	require.ErrorIs(t, useCase.Execute(t.Context(), battle(now)), discord.ErrRefused)
	pinged, err := store.StartPinged(t.Context(), "season-1/battle-1")
	require.NoError(t, err)
	require.False(t, pinged)

	poster.Err = nil
	require.NoError(t, useCase.Execute(t.Context(), battle(now)))
	assert.Len(t, poster.Posts, 1)
}
