package posts_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

// n tracks ranked from 1, each with the id of its excerpt, of 30s.
func chartOf(n int) []posts.Track {
	tracks := make([]posts.Track, 0, n)
	for rank := 1; rank <= n; rank++ {
		tracks = append(tracks, posts.Track{
			ID: fmt.Sprintf("clip-%d", rank), Excerpt: true, Title: fmt.Sprintf("Song %d", rank), Rank: rank, Seconds: 30,
		})
	}
	return tracks
}

// A clip that plays so many seconds.
func clipOf(t *testing.T, seconds float64) posts.Clip {
	t.Helper()

	written := rendered(1, "BEST")
	written.Seconds = seconds
	restored, err := written.Restore()
	require.NoError(t, err)
	return restored
}

type countingDice struct{ asked []int }

func (d *countingDice) IntN(n int) int {
	d.asked = append(d.asked, n)
	return n - 1
}

func TestATrendingTrackIsOneOfTheTop20MixedAloneOverTheMutedVideo(t *testing.T) {
	dice := &countingDice{}

	sound, err := posts.Soundtracks{Country: "US"}.Pick(chartOf(100), clipOf(t, 10), dice)

	require.NoError(t, err)
	assert.Equal(t, []int{20}, dice.asked)
	assert.Equal(t, posts.Sound{Track: chartOf(20)[19], Country: "US", TrackVolume: 100, OriginalVolume: 0}, sound)
}

func TestEachTrackOfTheTopCanBePickedInTheChartsOrder(t *testing.T) {
	for i := range 20 {
		sound, err := posts.Soundtracks{Country: "US"}.Pick(chartOf(100), clipOf(t, 10), posts.LoadedDice(i))

		require.NoError(t, err)
		assert.Equal(t, i+1, sound.Track.Rank)
	}
}

func TestAChartShorterThanTheTopIsPickedWhole(t *testing.T) {
	dice := &countingDice{}

	sound, err := posts.Soundtracks{Country: "US"}.Pick(chartOf(3), clipOf(t, 10), dice)

	require.NoError(t, err)
	assert.Equal(t, []int{3}, dice.asked)
	assert.Equal(t, "clip-3", sound.Track.ID)
}

// A track TikTok could not be sent is no track, and a rank the chart left out is its place in the list.
func TestATrackWithNoIDIsSkippedAndARankLeftOutIsItsPlace(t *testing.T) {
	chart := []posts.Track{
		{Title: "No id", Rank: 1, Seconds: 30},
		{ID: "track-2", Title: "Whole track", Seconds: 201},
		{ID: "clip-3", Excerpt: true, Rank: 3, Seconds: 30},
	}

	first, err := posts.Soundtracks{Country: "FR"}.Pick(chart, clipOf(t, 10), posts.LoadedDice(0))
	require.NoError(t, err)
	second, err := posts.Soundtracks{Country: "FR"}.Pick(chart, clipOf(t, 10), posts.LoadedDice(1))
	require.NoError(t, err)

	assert.Equal(t, posts.Track{ID: "track-2", Title: "Whole track", Rank: 2, Seconds: 201}, first.Track)
	assert.Equal(t, "FR", first.Country)
	assert.Equal(t, "clip-3", second.Track.ID)
}

// The chart ranks sound effects of a second beside the songs (a horn at 6 of the US chart on 2026-10-27), TikTok does
// not loop a track, and the anthem is muted under it: a track shorter than the clip would leave the rest of it silent.
func TestATrackShorterThanTheClipIsSkippedAndTheTopCountedWithoutIt(t *testing.T) {
	chart := chartOf(25)
	chart[2].Seconds, chart[5].Seconds = 15, 1

	for seconds, want := range map[float64]int{10: 21, 15: 21, 20.5: 22} {
		dice := &countingDice{}

		sound, err := posts.Soundtracks{Country: "US"}.Pick(chart, clipOf(t, seconds), dice)

		require.NoError(t, err)
		assert.Equal(t, []int{20}, dice.asked)
		assert.Equal(t, want, sound.Track.Rank, "a clip of %vs", seconds)
	}
	for i := range 20 {
		sound, err := posts.Soundtracks{Country: "US"}.Pick(chart, clipOf(t, 10), posts.LoadedDice(i))

		require.NoError(t, err)
		assert.NotEqual(t, 6, sound.Track.Rank)
	}
}

func TestAChartWithNothingToAttachIsAnError(t *testing.T) {
	for name, chart := range map[string][]posts.Track{
		"empty":        nil,
		"no id":        {{Title: "No id", Rank: 1, Seconds: 30}},
		"a horn":       {{ID: "clip-1", Excerpt: true, Title: "Breakout sound", Rank: 1, Seconds: 1}},
		"of no length": {{ID: "clip-1", Excerpt: true, Title: "Unsaid", Rank: 1}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := posts.Soundtracks{Country: "US"}.Pick(chart, clipOf(t, 12.37), posts.LoadedDice(0))

			require.ErrorIs(t, err, posts.ErrNoTrendingTrack)
			assert.Contains(t, err.Error(), "the US chart lists none that plays the clip's 12.4s")
		})
	}
}

func TestAClipOfUnknownLengthGetsNoTrack(t *testing.T) {
	_, err := posts.Soundtracks{Country: "US"}.Pick(chartOf(100), clipOf(t, 0), posts.LoadedDice(0))

	require.ErrorIs(t, err, posts.ErrNoTrendingTrack)
	assert.Contains(t, err.Error(), "unknown")
}

func TestAChartsCountryIsTwoLettersInUpperCase(t *testing.T) {
	for written, want := range map[string]string{"US": "US", "fr": "FR", "gb": "GB"} {
		code, err := posts.ChartCountryOf(written)

		require.NoError(t, err)
		assert.Equal(t, want, code)
	}
	spaced, err := posts.ChartCountryOf("  de ")
	require.NoError(t, err)
	assert.Equal(t, "DE", spaced)
	for _, written := range []string{"", "USA", "U", "1A", "é", "U-"} {
		_, err := posts.ChartCountryOf(written)

		require.ErrorIs(t, err, posts.ErrChartCountry, written)
	}
}
