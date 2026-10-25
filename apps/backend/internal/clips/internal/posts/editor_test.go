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
	editedAt     = time.Date(2026, 10, 24, 12, 17, 0, 0, time.UTC)
	palauEurope  = posts.StoryKey{Flag: "pw", Continent: "Europe"}
	franceEurope = posts.StoryKey{Flag: "fr", Continent: "Europe"}
	algeriaAfric = posts.StoryKey{Flag: "dz", Continent: "Africa"}
	brazilSouth  = posts.StoryKey{Flag: "br", Continent: "South America"}
	day5         = posts.StoryKey{Flag: "fr", Continent: "Europe", Angle: posts.DayWon, Round: "1.5"}
)

func editedShortlist(t *testing.T, stories ...posts.Story) posts.Shortlist {
	t.Helper()

	shortlist, err := posts.ShortlistOf(stories...)
	require.NoError(t, err)
	return shortlist
}

// Five stories: Palau, told in Europe 4 hours ago; France; Algeria, whose flag 2 of the last posts showed; Brazil; and
// Spain, with a teaser of its own.
func fiveStories(t *testing.T) (posts.Shortlist, []posts.Telling) {
	t.Helper()

	shortlist := editedShortlist(t,
		posts.Story{Title: "PALAU IS ATTACKING EUROPE", Key: palauEurope, Countries: []string{"pw"}},
		posts.Story{Title: "FRANCE VS GERMANY", Key: franceEurope, Countries: []string{"fr", "de"}},
		posts.Story{Title: "ALGERIA STRIKES BACK", Key: algeriaAfric, Countries: []string{"dz"}},
		posts.Story{Title: "BRAZIL IS INVADING ARGENTINA", Key: brazilSouth, Countries: []string{"br", "ar"}},
		posts.Story{
			Title: "SPAIN IS ATTACKING AFRICA", Key: posts.StoryKey{Flag: "es", Continent: "Africa"}, Countries: []string{"es", "dz"},
			Teasers: []string{"SPAIN TAKES AFRICA"},
		},
	)
	told := []posts.Telling{
		{Key: palauEurope, Title: "PALAU IS ATTACKING EUROPE", At: editedAt.Add(-4 * time.Hour)},
		{Key: algeriaAfric, Title: "ALGERIA VS MOROCCO", At: editedAt.Add(-5 * 24 * time.Hour)},
		{Key: posts.StoryKey{Flag: "ma", Continent: "Africa"}, Title: "MOROCCO IS FALLING", Countries: []string{"ma", "dz"}, At: editedAt.Add(-2 * time.Hour)},
		{Key: posts.StoryKey{Flag: "tn", Continent: "Africa"}, Title: "ALGERIA IS EVERYWHERE", Countries: []string{"tn", "dz"}, At: editedAt.Add(-3 * time.Hour)},
	}
	return shortlist, told
}

func TestTheEditorIsOfferedOnlyTheStoriesTheRulesWouldTell(t *testing.T) {
	shortlist, told := fiveStories(t)

	offer := shortlist.Offer(2, told, editedAt)

	assert.Equal(t, []posts.Pick{2, 4, 5}, offer.Stories.Picks(), "Palau told, Algeria in 2 of the last posts")
	assert.Equal(t, 2, offer.Slots())
	assert.True(t, offer.Open())
	assert.Equal(t, 3, offer.Shown["dz"])
	assert.Equal(t, 1, offer.Shown["pw"])
	assert.Equal(t, []string{"MOROCCO IS FALLING", "ALGERIA IS EVERYWHERE", "PALAU IS ATTACKING EUROPE"}, titlesOf(offer.Recent),
		"the last 3 days, newest first")
}

func TestADayWonIsToldFirstAndLeavesTheEditorTheRest(t *testing.T) {
	shortlist := editedShortlist(t,
		posts.Story{Title: "FRANCE WINS DAY 5", Key: day5},
		posts.Story{Title: "BRAZIL IS INVADING ARGENTINA", Key: brazilSouth},
	)

	assert.Equal(t, 1, shortlist.Offer(2, nil, editedAt).Slots())
	assert.Equal(t, []posts.Pick{1}, shortlist.Offer(2, nil, editedAt).First.Picks())
	assert.False(t, shortlist.Offer(1, nil, editedAt).Open(), "a run of one day won asks nobody")
	assert.False(t, shortlist.Offer(0, nil, editedAt).Open(), "a full day asks nobody")
	assert.False(t, editedShortlist(t).Offer(2, nil, editedAt).Open(), "a quiet run asks nobody")
}

func TestTheRunFollowsTheEditorsOrderAndHeadlines(t *testing.T) {
	shortlist, told := fiveStories(t)
	offer := shortlist.Offer(2, told, editedAt)

	choice, headlines, err := shortlist.Edited(offer, posts.Edition{Edits: []posts.Edit{
		{Pick: 4, Headline: "  brazil storms  argentina! "},
		{Pick: 2, Headline: "FRANCE VS GERMANY"},
	}}, told, editedAt)

	require.NoError(t, err)
	assert.Equal(t, []posts.Pick{2, 4}, choice.Chosen.Picks())
	assert.Equal(t, []string{"FRANCE VS GERMANY", "BRAZIL STORMS ARGENTINA!"}, titlesOf(choice.Chosen))
	assert.Equal(t, map[posts.Pick]string{4: "BRAZIL STORMS ARGENTINA!"}, headlines, "only the headlines it changed")
	assert.Equal(t, []posts.Pick{1}, choice.Retold.Picks())
}

func TestTheRulesStillCountTheRunAndFillWhatTheEditorLeftOut(t *testing.T) {
	shortlist, told := fiveStories(t)
	offer := shortlist.Offer(2, told, editedAt)

	choice, _, err := shortlist.Edited(offer, posts.Edition{Edits: []posts.Edit{{Pick: 5, Headline: "SPAIN MARCHES ON AFRICA"}}}, told, editedAt)

	require.NoError(t, err)
	assert.Len(t, choice.Chosen, 2, "as many as the rules would make")
	assert.Contains(t, choice.Chosen.Picks(), posts.Pick(5))
}

// A flag the last posts showed once takes one more clip: the editor's second story of it gives way to the next.
func TestAStoryOfTheEditorsWhoseFlagTheRunFilledGivesWay(t *testing.T) {
	shortlist := editedShortlist(t,
		posts.Story{Title: "PALAU IS ATTACKING EUROPE", Key: palauEurope},
		posts.Story{Title: "PALAU IS INVADING BRAZIL", Key: posts.StoryKey{Flag: "pw", Continent: "South America"}},
		posts.Story{Title: "FRANCE VS GERMANY", Key: franceEurope},
	)
	told := []posts.Telling{{Key: posts.StoryKey{Flag: "pw", Continent: "Asia"}, Title: "PALAU IS EVERYWHERE", At: editedAt.Add(-time.Hour)}}
	offer := shortlist.Offer(2, told, editedAt)

	choice, _, err := shortlist.Edited(offer, posts.Edition{Edits: []posts.Edit{
		{Pick: 1, Headline: "PALAU TAKES EUROPE"}, {Pick: 2, Headline: "PALAU TAKES BRAZIL"}, {Pick: 3, Headline: "FRANCE VS GERMANY"},
	}}, told, editedAt)

	require.NoError(t, err)
	assert.Equal(t, []posts.Pick{1, 3}, choice.Chosen.Picks())
}

func TestADayWonGoesFirstWhateverTheEditorSays(t *testing.T) {
	shortlist := editedShortlist(t,
		posts.Story{Title: "FRANCE WINS DAY 5", Key: day5},
		posts.Story{Title: "BRAZIL IS INVADING ARGENTINA", Key: brazilSouth},
		posts.Story{Title: "ALGERIA STRIKES BACK", Key: algeriaAfric},
	)
	offer := shortlist.Offer(2, nil, editedAt)

	choice, _, err := shortlist.Edited(offer, posts.Edition{Edits: []posts.Edit{{Pick: 3, Headline: "ALGERIA FIGHTS BACK"}}}, nil, editedAt)

	require.NoError(t, err)
	assert.Equal(t, []posts.Pick{1, 3}, choice.Chosen.Picks())
}

func TestAnEditionThatStraysFromTheRulesDoesNotHold(t *testing.T) {
	shortlist, told := fiveStories(t)
	offer := shortlist.Offer(2, told, editedAt)

	for name, edits := range map[string][]posts.Edit{
		"no story":                     nil,
		"a story told lately":          {{Pick: 1, Headline: "PALAU AGAIN"}},
		"a flag the last posts filled": {{Pick: 3, Headline: "ALGERIA RISES"}},
		"a story not in the shortlist": {{Pick: 9, Headline: "NOWHERE"}},
		"a story twice":                {{Pick: 2, Headline: "A"}, {Pick: 2, Headline: "B"}},
		"no headline":                  {{Pick: 2, Headline: "  "}},
		"a headline too long":          {{Pick: 2, Headline: strings.Repeat("FRANCE ", 9)}},
		"an emoji":                     {{Pick: 2, Headline: "FRANCE 🔥 GERMANY"}},
		"a flag of the generator":      {{Pick: 2, Headline: "--PLAN"}},
		"a quote":                      {{Pick: 2, Headline: "FRANCE \"VS\" GERMANY"}},
		"a headline of the last week":  {{Pick: 2, Headline: "algeria vs morocco"}},
		"a teaser of its caption":      {{Pick: 5, Headline: "Spain takes Africa"}},
		"one headline for two":         {{Pick: 2, Headline: "WAR IN THE WEST"}, {Pick: 4, Headline: "WAR IN THE WEST"}},
	} {
		t.Run(name, func(t *testing.T) {
			_, _, err := shortlist.Edited(offer, posts.Edition{Edits: edits}, told, editedAt)

			require.ErrorIs(t, err, posts.ErrInvalidEdition)
		})
	}
}

func TestAHeadlineInTheCountrysOwnLanguageHolds(t *testing.T) {
	shortlist := editedShortlist(t, posts.Story{
		Title: "POLSKA: 14 PUNKTÓW STRATY", Key: posts.StoryKey{Flag: "pl", Continent: "Europe", Angle: posts.CallUp, Round: "1.5"},
	})

	choice, headlines, err := shortlist.Edited(shortlist.Offer(1, nil, editedAt),
		posts.Edition{Edits: []posts.Edit{{Pick: 1, Headline: "Polska, do boju!"}}}, nil, editedAt)

	require.NoError(t, err)
	assert.Equal(t, "POLSKA, DO BOJU!", headlines[1])
	assert.Equal(t, []string{"POLSKA, DO BOJU!"}, titlesOf(choice.Chosen))
}

func titlesOf[T posts.Story | posts.Telling](items []T) []string {
	titles := make([]string, 0, len(items))
	for _, item := range items {
		switch of := any(item).(type) {
		case posts.Story:
			titles = append(titles, of.Title)
		case posts.Telling:
			titles = append(titles, of.Title)
		}
	}
	return titles
}
