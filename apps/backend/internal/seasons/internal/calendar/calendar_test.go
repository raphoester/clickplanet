package calendar_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
)

var (
	seasonZeroEnds = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	seasonOneEnds  = time.Date(2026, 12, 31, 23, 0, 0, 0, time.UTC)
)

func twoSeasons() calendar.Config {
	return calendar.Config{List: []calendar.Entry{
		{Number: 0, EndsAt: seasonZeroEnds, Finale: 2 * time.Hour},
		{Number: 1, EndsAt: seasonOneEnds, Finale: 3 * time.Hour},
	}}
}

func TestTheCurrentSeasonIsTheFirstThatHasNotEnded(t *testing.T) {
	seasons := calendar.New(twoSeasons())

	season, ok := seasons.Current(seasonZeroEnds.Add(-time.Second))
	require.True(t, ok)
	assert.Equal(t, calendar.Season{
		Number:         0,
		FinaleStartsAt: time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC),
		EndsAt:         seasonZeroEnds,
	}, season)

	season, ok = seasons.Current(seasonZeroEnds.Add(time.Second))
	require.True(t, ok)
	assert.Equal(t, calendar.Season{
		Number:         1,
		FinaleStartsAt: time.Date(2026, 12, 31, 20, 0, 0, 0, time.UTC),
		EndsAt:         seasonOneEnds,
	}, season)

	season, ok = seasons.Current(seasonOneEnds.Add(-time.Second))
	require.True(t, ok)
	assert.Equal(t, calendar.Number(1), season.Number)

	_, ok = seasons.Current(seasonOneEnds.Add(time.Second))
	assert.False(t, ok)
}

func TestASeasonIsOverAtTheInstantItEnds(t *testing.T) {
	season, ok := calendar.New(twoSeasons()).Current(seasonZeroEnds)

	require.True(t, ok)
	assert.Equal(t, calendar.Number(1), season.Number)
}

func TestAnEmptyListHasNoSeason(t *testing.T) {
	require.NoError(t, calendar.Config{}.Validate())

	_, ok := calendar.New(calendar.Config{}).Current(seasonZeroEnds)

	assert.False(t, ok)
}

func TestTwoSeasonsAreAccepted(t *testing.T) {
	require.NoError(t, twoSeasons().Validate())
}

func TestNumbersThatDoNotCountUpFromZeroAreRefused(t *testing.T) {
	config := twoSeasons()
	config.List[0].Number = 1
	config.List[1].Number = 2
	require.ErrorContains(t, config.Validate(), "list[0].number is 1")

	config = twoSeasons()
	config.List[1].Number = 2
	require.ErrorContains(t, config.Validate(), "list[1].number is 2")
}

func TestEndsThatDoNotGoUpAreRefused(t *testing.T) {
	config := twoSeasons()
	config.List[1].EndsAt = seasonZeroEnds
	require.ErrorContains(t, config.Validate(), "list[1].endsAt")

	config = twoSeasons()
	config.List[0].EndsAt = time.Time{}
	require.ErrorContains(t, config.Validate(), "list[0].endsAt")
}

func TestAFinaleNotShorterThanItsSeasonIsRefused(t *testing.T) {
	config := twoSeasons()
	config.List[1].Finale = seasonOneEnds.Sub(seasonZeroEnds)
	require.ErrorContains(t, config.Validate(), "list[1].finale")

	config.List[1].Finale -= time.Second
	require.NoError(t, config.Validate())
}

func TestAFinaleOfNothingIsRefused(t *testing.T) {
	config := twoSeasons()
	config.List[0].Finale = 0
	require.ErrorContains(t, config.Validate(), "list[0].finale")

	config = twoSeasons()
	config.List[0].Finale = -time.Hour
	require.ErrorContains(t, config.Validate(), "list[0].finale")
}
