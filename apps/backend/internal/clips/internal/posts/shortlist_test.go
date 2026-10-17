package posts_test

import (
	"strings"
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

func toldAgo(key posts.StoryKey, title string, ago time.Duration) posts.Telling {
	return posts.Telling{Key: key, Title: title, At: theNightOfOct14.Add(-ago)}
}

func TestAShortlistNumbersItsStoriesFromOneBestFirst(t *testing.T) {
	assert.Equal(t, []posts.Pick{1, 2, 3, 4, 5}, shortlistOf(t, written...).Picks())
}

func TestTheBestStoriesAreTheFirstOfTheShortlist(t *testing.T) {
	choice := shortlistOf(t, written[0], written[2], written[4]).Best(2, nil, theNightOfOct14)

	assert.Equal(t, []string{"ALGERIA STRIKES BACK", "PALESTINE IS KICKING ROMANIA OUT OF KAZAKHSTAN"}, titles(choice.Chosen))
	assert.Equal(t, []posts.Pick{1, 2}, choice.Chosen.Picks())
	assert.Empty(t, choice.Retold)
	assert.Empty(t, choice.Passed)
}

func TestOneRunTellsOneStoryOfAFlagOnAContinent(t *testing.T) {
	choice := shortlistOf(t, written...).Best(2, nil, theNightOfOct14)

	assert.Equal(t, []string{"ALGERIA STRIKES BACK", "PALESTINE IS KICKING ROMANIA OUT OF KAZAKHSTAN"}, titles(choice.Chosen))
	assert.Equal(t, []posts.Pick{1, 3}, choice.Chosen.Picks())
	assert.Equal(t, []string{"ALGERIA VS FRANCE"}, titles(choice.Retold))
}

func TestAStoryToldInTheLastThreeDaysIsNotToldAgain(t *testing.T) {
	told := []posts.Telling{
		toldAgo(algeriaInAfrica, "ALGERIA IS INVADING LIBYA", 71*time.Hour),
		toldAgo(palestineInAsia, "PALESTINE STRIKES BACK", 2*time.Hour),
	}

	choice := shortlistOf(t, written...).Best(2, told, theNightOfOct14)

	assert.Equal(t, []string{"ALGERIA IS INVADING INDIA", "FRANCE IS ATTACKING EUROPE"}, titles(choice.Chosen))
	assert.Equal(t, []string{"ALGERIA STRIKES BACK", "ALGERIA VS FRANCE", "PALESTINE IS KICKING ROMANIA OUT OF KAZAKHSTAN"},
		titles(choice.Retold))
}

func TestAStoryScheduledForLaterIsToldAlready(t *testing.T) {
	choice := shortlistOf(t, written...).Best(1, []posts.Telling{toldAgo(algeriaInAfrica, "ALGERIA IS INVADING LIBYA", -3*time.Hour)}, theNightOfOct14)

	assert.Equal(t, []string{"PALESTINE IS KICKING ROMANIA OUT OF KAZAKHSTAN"}, titles(choice.Chosen))
}

func TestAStoryToldMoreThanThreeDaysAgoCanBeToldAgain(t *testing.T) {
	before := []posts.Telling{toldAgo(algeriaInAfrica, "ALGERIA IS INVADING LIBYA", 72*time.Hour+time.Second)}
	exactly := []posts.Telling{toldAgo(algeriaInAfrica, "ALGERIA IS INVADING LIBYA", 72*time.Hour)}

	algeria := shortlistOf(t, written[0], written[3])

	assert.Equal(t, []string{"ALGERIA STRIKES BACK"}, titles(algeria.Best(1, before, theNightOfOct14).Chosen))
	assert.Equal(t, []string{"ALGERIA IS INVADING INDIA"}, titles(algeria.Best(1, exactly, theNightOfOct14).Chosen))
}

func TestAHeadlineToldInTheLastWeekIsNotToldAgainOnAnotherContinent(t *testing.T) {
	told := []posts.Telling{toldAgo(algeriaInAsia, "Algeria strikes back ", 6*24*time.Hour)}

	choice := shortlistOf(t, written[:2]...).Best(1, told, theNightOfOct14)

	assert.Equal(t, []string{"ALGERIA VS FRANCE"}, titles(choice.Chosen))
	assert.Equal(t, []string{"ALGERIA STRIKES BACK"}, titles(choice.Retold))
}

func TestAHeadlineToldMoreThanAWeekAgoCanBeToldAgain(t *testing.T) {
	before := []posts.Telling{toldAgo(algeriaInAsia, "ALGERIA STRIKES BACK", 7*24*time.Hour+time.Second)}
	exactly := []posts.Telling{toldAgo(algeriaInAsia, "ALGERIA STRIKES BACK", 7*24*time.Hour)}

	assert.Equal(t, []string{"ALGERIA STRIKES BACK"}, titles(shortlistOf(t, written[:2]...).Best(1, before, theNightOfOct14).Chosen))
	assert.Equal(t, []string{"ALGERIA VS FRANCE"}, titles(shortlistOf(t, written[:2]...).Best(1, exactly, theNightOfOct14).Chosen))
}

func TestTheHistoryReachesBackAWeek(t *testing.T) {
	assert.Equal(t, theNightOfOct14.Add(-7*24*time.Hour), posts.ToldSince(theNightOfOct14))
}

func TestAFlagTheLastPostsToldRanksAfterTheFlagsTheyDidNot(t *testing.T) {
	told := []posts.Telling{toldAgo(posts.StoryKey{Flag: "dz", Continent: "Europe"}, "ALGERIA IS ATTACKING EUROPE", time.Hour)}

	choice := shortlistOf(t, written...).Best(1, told, theNightOfOct14)

	assert.Equal(t, []string{"PALESTINE IS KICKING ROMANIA OUT OF KAZAKHSTAN"}, titles(choice.Chosen))
	assert.Equal(t, []string{"ALGERIA STRIKES BACK", "ALGERIA VS FRANCE"}, titles(choice.Passed))
}

func TestAFlagInTwoOfTheLastPostsWaits(t *testing.T) {
	told := []posts.Telling{
		toldAgo(posts.StoryKey{Flag: "dz", Continent: "Europe"}, "ALGERIA IS ATTACKING EUROPE", time.Hour),
		toldAgo(posts.StoryKey{Flag: "dz", Continent: "Antarctica"}, "ALGERIA IS INVADING ANTARCTICA", 3*time.Hour),
	}

	choice := shortlistOf(t, written...).Best(6, told, theNightOfOct14)

	assert.Equal(t, []string{"PALESTINE IS KICKING ROMANIA OUT OF KAZAKHSTAN", "FRANCE IS ATTACKING EUROPE"}, titles(choice.Chosen))
	assert.Equal(t, []string{"ALGERIA STRIKES BACK", "ALGERIA VS FRANCE", "ALGERIA IS INVADING INDIA"}, titles(choice.Passed))
}

func TestOnlyTheLastSixPostsCountAFlag(t *testing.T) {
	flag := func(code string, hours int) posts.Telling {
		return toldAgo(posts.StoryKey{Flag: code, Continent: "Europe"}, strings.ToUpper(code)+" IS ATTACKING EUROPE", time.Duration(hours)*time.Hour)
	}
	told := []posts.Telling{flag("ps", 1), flag("fr", 2), flag("de", 3), flag("it", 4), flag("es", 5), flag("pt", 6), flag("dz", 7), flag("dz", 8)}

	choice := shortlistOf(t, written[0], written[2], written[4]).Best(1, told, theNightOfOct14)

	assert.Equal(t, []string{"ALGERIA STRIKES BACK"}, titles(choice.Chosen))
}

func TestARunTellsAnotherFlagAfterTheOneItChose(t *testing.T) {
	stories := []posts.Story{
		{Title: "PALAU IS TAKING OVER THE WORLD", Key: posts.StoryKey{Flag: "pw", Continent: "the world"}},
		{Title: "PALAU IS KICKING FRANCE OUT OF AFGHANISTAN AND UZBEKISTAN", Key: posts.StoryKey{Flag: "pw", Continent: "South Asia"}},
		{Title: "FRANCE STRIKES BACK", Key: franceInEurope},
	}

	choice := shortlistOf(t, stories...).Best(2, nil, theNightOfOct14)

	assert.Equal(t, []posts.Pick{1, 3}, choice.Chosen.Picks())
	assert.Equal(t, []posts.Pick{2}, choice.Passed.Picks())
}

func TestAShortlistOfOneFlagIsToldTwiceInARun(t *testing.T) {
	stories := []posts.Story{
		{Title: "PALAU IS TAKING OVER THE WORLD", Key: posts.StoryKey{Flag: "pw", Continent: "the world"}},
		{Title: "PALAU IS ATTACKING EAST ASIA", Key: posts.StoryKey{Flag: "pw", Continent: "East Asia"}},
		{Title: "PALAU VS FRANCE", Key: posts.StoryKey{Flag: "pw", Continent: "Europe"}},
	}

	choice := shortlistOf(t, stories...).Best(6, nil, theNightOfOct14)

	assert.Equal(t, []posts.Pick{1, 2}, choice.Chosen.Picks())
	assert.Equal(t, []posts.Pick{3}, choice.Passed.Picks())
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

func TestTheNightOfOct17RetellsNothingAndTellsFlagsTheLastDayDidNot(t *testing.T) {
	key := func(flag, continent string) posts.StoryKey { return posts.StoryKey{Flag: flag, Continent: continent} }
	at := func(day, hour, minute int) time.Time { return time.Date(2026, 10, day, hour, minute, 0, 0, time.UTC) }
	told := []posts.Telling{
		{Key: key("ma", "South America"), Title: "MOROCCO IS INVADING BRAZIL", At: at(14, 16, 59)},
		{Key: key("ma", "Europe"), Title: "MOROCCO IS ATTACKING EUROPE", At: at(14, 18, 59)},
		{Key: key("dz", "Africa"), Title: "ALGERIA IS INVADING LIBYA", At: at(15, 5, 15)},
		{Key: key("fr", "Africa"), Title: "FRANCE IS KICKING MOROCCO OUT OF LIBYA", At: at(15, 7, 15)},
		{Key: key("pw", "the world"), Title: "PALAU IS TAKING OVER THE WORLD", At: at(15, 17, 1)},
		{Key: key("pw", "South Asia"), Title: "PALAU IS KICKING FRANCE OUT OF AFGHANISTAN AND UZBEKISTAN", At: at(15, 19, 1)},
		{Key: key("pw", "Europe"), Title: "PALAU VS FRANCE", At: at(16, 5, 24)},
		{Key: key("pw", "East Asia"), Title: "PALAU IS ATTACKING EAST ASIA", At: at(16, 7, 24)},
		{Key: key("dz", "Africa"), Title: "ALGERIA STRIKES BACK", At: at(16, 16, 55)},
		{Key: key("ps", "Middle East"), Title: "PALESTINE STRIKES BACK", At: at(16, 18, 55)},
	}
	shortlist := shortlistOf(t,
		posts.Story{Title: "PALAU IS TAKING OVER THE WORLD", Key: key("pw", "the world")},
		posts.Story{Title: "MOROCCO GETS KICKED OUT OF EUROPE", Key: key("ma", "Europe")},
		posts.Story{Title: "PALAU IS ATTACKING EUROPE", Key: key("pw", "Europe")},
		posts.Story{Title: "FRANCE STRIKES BACK", Key: key("fr", "Europe")},
		posts.Story{Title: "PALAU IS KICKING CROATIA OUT OF CHINA", Key: key("pw", "East Asia")},
		posts.Story{Title: "FRANCE VS PALAU", Key: key("fr", "Europe")},
		posts.Story{Title: "ALGERIA IS KICKING BELGIUM OUT OF TURKEY", Key: key("dz", "Middle East")},
		posts.Story{Title: "VATICAN CITY VS PALAU", Key: key("va", "Europe")},
		posts.Story{Title: "FRANCE VS PALAU", Key: key("fr", "Europe")},
		posts.Story{Title: "PALAU VS FRANCE", Key: key("pw", "Europe")},
		posts.Story{Title: "THE UK STRIKES BACK", Key: key("gb", "Europe")},
		posts.Story{Title: "PALESTINE IS INVADING ANTARCTICA", Key: key("ps", "Antarctica")},
		posts.Story{Title: "KOSOVO IS KICKING HUNGARY OUT OF FINLAND", Key: key("xk", "Europe")},
		posts.Story{Title: "KOSOVO IS INVADING GERMANY", Key: key("xk", "Europe")},
		posts.Story{Title: "GERMANY IS INVADING POLAND", Key: key("de", "Europe")},
	)

	choice := shortlist.Best(2, told, at(17, 1, 0))

	assert.Equal(t, []string{"FRANCE STRIKES BACK", "VATICAN CITY VS PALAU"}, titles(choice.Chosen))
	assert.Equal(t, []posts.Pick{1, 2, 3, 5, 6, 9, 10}, choice.Retold.Picks())
	assert.Equal(t, []string{"ALGERIA IS KICKING BELGIUM OUT OF TURKEY"}, titles(choice.Passed))
}
