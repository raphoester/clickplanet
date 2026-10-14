package posts_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

var (
	algeriaInAfrica = posts.StoryKey{Flag: "dz", Continent: "Africa"}
	algeriaInAsia   = posts.StoryKey{Flag: "dz", Continent: "South Asia"}
	palestineInAsia = posts.StoryKey{Flag: "ps", Continent: "Central Asia"}
	franceInEurope  = posts.StoryKey{Flag: "fr", Continent: "Europe"}
	theNightOfOct14 = time.Date(2026, 10, 14, 0, 57, 0, 0, time.UTC)
	written         = []posts.Story{
		{Title: "ALGERIA STRIKES BACK", Key: algeriaInAfrica},
		{Title: "ALGERIA VS FRANCE", Key: algeriaInAfrica},
		{Title: "PALESTINE IS KICKING ROMANIA OUT OF KAZAKHSTAN", Key: palestineInAsia},
		{Title: "ALGERIA IS INVADING INDIA", Key: algeriaInAsia},
		{Title: "FRANCE IS ATTACKING EUROPE", Key: franceInEurope},
	}
)

func shortlistOf(t *testing.T, stories ...posts.Story) posts.Shortlist {
	t.Helper()

	shortlist, err := posts.ShortlistOf(stories...)
	require.NoError(t, err)
	return shortlist
}

func titles(shortlist posts.Shortlist) []string {
	told := make([]string, 0, len(shortlist))
	for _, story := range shortlist {
		told = append(told, story.Title)
	}
	return told
}

func TestAShortlistNumbersItsStoriesFromOneBestFirst(t *testing.T) {
	assert.Equal(t, []posts.Pick{1, 2, 3, 4, 5}, shortlistOf(t, written...).Picks())
}

func TestTheBestStoriesAreTheFirstOfTheShortlist(t *testing.T) {
	choice := shortlistOf(t, written[0], written[2], written[4]).Best(2, nil, theNightOfOct14)

	assert.Equal(t, []string{"ALGERIA STRIKES BACK", "PALESTINE IS KICKING ROMANIA OUT OF KAZAKHSTAN"}, titles(choice.Chosen))
	assert.Equal(t, []posts.Pick{1, 2}, choice.Chosen.Picks())
	assert.Empty(t, choice.Retold)
}

func TestOneRunTellsOneStoryOfAFlagOnAContinent(t *testing.T) {
	choice := shortlistOf(t, written...).Best(2, nil, theNightOfOct14)

	assert.Equal(t, []string{"ALGERIA STRIKES BACK", "PALESTINE IS KICKING ROMANIA OUT OF KAZAKHSTAN"}, titles(choice.Chosen))
	assert.Equal(t, []posts.Pick{1, 3}, choice.Chosen.Picks())
	assert.Equal(t, []string{"ALGERIA VS FRANCE"}, titles(choice.Retold))
}

func TestAStoryToldInTheLastDayIsNotToldAgain(t *testing.T) {
	told := []posts.Telling{
		{Key: algeriaInAfrica, At: time.Date(2026, 10, 13, 21, 9, 0, 0, time.UTC)},
		{Key: palestineInAsia, At: time.Date(2026, 10, 13, 23, 9, 0, 0, time.UTC)},
	}

	choice := shortlistOf(t, written...).Best(2, told, theNightOfOct14)

	assert.Equal(t, []string{"ALGERIA IS INVADING INDIA", "FRANCE IS ATTACKING EUROPE"}, titles(choice.Chosen))
	assert.Equal(t, []string{"ALGERIA STRIKES BACK", "ALGERIA VS FRANCE", "PALESTINE IS KICKING ROMANIA OUT OF KAZAKHSTAN"},
		titles(choice.Retold))
}

func TestAStoryScheduledForLaterIsToldAlready(t *testing.T) {
	choice := shortlistOf(t, written...).Best(1, []posts.Telling{{Key: algeriaInAfrica, At: theNightOfOct14.Add(3 * time.Hour)}}, theNightOfOct14)

	assert.Equal(t, []string{"PALESTINE IS KICKING ROMANIA OUT OF KAZAKHSTAN"}, titles(choice.Chosen))
}

func TestAStoryToldMoreThanADayAgoCanBeToldAgain(t *testing.T) {
	dayBefore := []posts.Telling{{Key: algeriaInAfrica, At: posts.ToldSince(theNightOfOct14).Add(-time.Second)}}
	exactlyADay := []posts.Telling{{Key: algeriaInAfrica, At: posts.ToldSince(theNightOfOct14)}}

	assert.Equal(t, []string{"ALGERIA STRIKES BACK"}, titles(shortlistOf(t, written...).Best(1, dayBefore, theNightOfOct14).Chosen))
	assert.Equal(t, []string{"PALESTINE IS KICKING ROMANIA OUT OF KAZAKHSTAN"}, titles(shortlistOf(t, written...).Best(1, exactlyADay, theNightOfOct14).Chosen))
}

func TestAShortlistWithFewerStoriesGivesAllItHas(t *testing.T) {
	assert.Equal(t, []posts.Pick{1}, shortlistOf(t, written[0]).Best(2, nil, theNightOfOct14).Chosen.Picks())
}

func TestAQuietHalfDayGivesNoPick(t *testing.T) {
	assert.Empty(t, shortlistOf(t).Best(2, nil, theNightOfOct14).Chosen.Picks())
	assert.Empty(t, shortlistOf(t, written...).Best(0, nil, theNightOfOct14).Chosen.Picks())
}

func TestAStoryTheGeneratorGaveNoKeyIsRefused(t *testing.T) {
	_, err := posts.ShortlistOf(written[0], posts.Story{Title: "PORTUGAL STRIKES BACK", Key: posts.StoryKey{Flag: "pt"}})

	require.ErrorIs(t, err, posts.ErrInvalidStory)
	assert.ErrorContains(t, err, "PORTUGAL STRIKES BACK")
}
