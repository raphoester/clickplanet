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

type seasons struct{ current, next *calendar.Season }

func (s seasons) Execute(context.Context) (current, next *calendar.Season) {
	return s.current, s.next
}

func getSeason(t *testing.T, s seasons) *seasonsv1.GetSeasonResponse {
	t.Helper()

	res, err := get_season_handler.New(s).GetSeason(t.Context(), connect.NewRequest(&seasonsv1.GetSeasonRequest{}))
	require.NoError(t, err)
	return res.Msg
}

func TestTheSeasonIsAnsweredInMilliseconds(t *testing.T) {
	season := getSeason(t, seasons{current: &calendar.Season{
		Number:         3,
		FinaleStartsAt: time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC),
		EndsAt:         time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC),
	}}).GetSeason()

	require.NotNil(t, season)
	assert.Equal(t, uint32(3), season.GetNumber())
	assert.Equal(t, int64(1793480400000), season.GetFinaleStartsAtUnixMs())
	assert.Equal(t, int64(1793487600000), season.GetEndsAtUnixMs())
	assert.Zero(t, season.GetStartsAtUnixMs(), "a season that started before anything has no start")
}

func TestBetweenTwoSeasonsOnlyTheNextIsAnsweredWithItsStart(t *testing.T) {
	res := getSeason(t, seasons{next: &calendar.Season{
		Number:         1,
		StartsAt:       time.Date(2026, 11, 1, 19, 0, 0, 0, time.UTC),
		FinaleStartsAt: time.Date(2026, 12, 5, 21, 0, 0, 0, time.UTC),
		EndsAt:         time.Date(2026, 12, 5, 23, 0, 0, 0, time.UTC),
	}})

	assert.Nil(t, res.GetSeason())
	require.NotNil(t, res.GetNext())
	assert.Equal(t, uint32(1), res.GetNext().GetNumber())
	assert.Equal(t, time.Date(2026, 11, 1, 19, 0, 0, 0, time.UTC).UnixMilli(), res.GetNext().GetStartsAtUnixMs())
}

func TestNoSeasonIsAnEmptyAnswer(t *testing.T) {
	res := getSeason(t, seasons{})

	assert.Nil(t, res.GetSeason())
	assert.Nil(t, res.GetNext())
}
