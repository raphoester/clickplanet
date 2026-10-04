package get_season_handler_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_season_handler"
)

type currentSeason calendar.Season

func (s currentSeason) Execute(context.Context) (calendar.Season, bool) {
	return calendar.Season(s), true
}

type noSeason struct{}

func (noSeason) Execute(context.Context) (calendar.Season, bool) {
	return calendar.Season{}, false
}

func TestTheSeasonIsAnsweredInMilliseconds(t *testing.T) {
	res, err := get_season_handler.New(currentSeason{
		Number:         3,
		FinaleStartsAt: time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC),
		EndsAt:         time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC),
	}).GetSeason(t.Context(), connect.NewRequest(&seasonsv1.GetSeasonRequest{}))
	require.NoError(t, err)

	season := res.Msg.GetSeason()
	require.NotNil(t, season)
	assert.Equal(t, uint32(3), season.GetNumber())
	assert.Equal(t, int64(1793480400000), season.GetFinaleStartsAtUnixMs())
	assert.Equal(t, int64(1793487600000), season.GetEndsAtUnixMs())
}

func TestNoSeasonIsAnEmptyAnswer(t *testing.T) {
	res, err := get_season_handler.New(noSeason{}).
		GetSeason(t.Context(), connect.NewRequest(&seasonsv1.GetSeasonRequest{}))
	require.NoError(t, err)

	assert.Nil(t, res.Msg.GetSeason())
}
