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
	battle = posts.Story{Front: 1, Kind: posts.Battle, Attacker: "fr", Rival: "tr", TakenFrom: []string{"tr"}, Countries: []string{"tr", "sy"}, Headline: "France vs Turkey", Teasers: []string{"THE WAR FOR TURKEY"}}
	india  = posts.Story{Front: 1, Kind: posts.Rout, Attacker: "fr", TakenFrom: []string{"dz"}, Countries: []string{"in"}, Headline: "ALGERIA LOSES ALL OF INDIA", Teasers: []string{"FRANCE TAKES INDIA"}}
	tour   = posts.Story{Tour: 1, Headline: "FRANCE IS EVERYWHERE", Teasers: []string{"NOBODY CAN STOP FRANCE"}}
)

func told(key posts.StoryKey, title string, ago time.Duration, countries ...string) posts.Telling {
	return posts.Telling{Key: key, Title: title, At: foundAt.Add(-ago), Countries: countries}
}

func TestTheOfferIsTheFindingsWholeAndTheLastPostsToVaryOn(t *testing.T) {
	found := findings(t)
	posted := []posts.Telling{
		told(posts.StoryKey{Flag: "tr", Continent: "Middle East"}, "TURKEY KICKS ALGERIA OUT OF IRAN", 28*time.Hour, "ir", "tr", "dz"),
		told(posts.StoryKey{Flag: "bg", Continent: "East Asia"}, "BULGARIA IS INVADING MONGOLIA", 4*time.Hour, "mn", "bg", "fr"),
		told(posts.StoryKey{Flag: "es", Continent: "Africa"}, "SPAIN IS SWALLOWING MOROCCO", 5*24*time.Hour, "ma", "es"),
	}

	offer := found.Offer(2, posted, foundAt)

	assert.Equal(t, found, offer.Findings, "nothing is kept from the editor")
	assert.Equal(t, []int{1}, offer.First)
	assert.Equal(t, 1, offer.Slots())
	assert.True(t, offer.Open())
	assert.Equal(t, []string{"BULGARIA IS INVADING MONGOLIA", "TURKEY KICKS ALGERIA OUT OF IRAN"}, titlesOf(offer.Recent), "the last 3 days, newest first")
	assert.Equal(t, map[string]int{"ir": 1, "tr": 1, "dz": 1, "mn": 1, "bg": 1, "fr": 1, "ma": 1, "es": 1}, offer.Shown)
}

func titlesOf(tellings []posts.Telling) []string {
	titles := make([]string, 0, len(tellings))
	for _, telling := range tellings {
		titles = append(titles, telling.Title)
	}
	return titles
}

func TestASeasonalStoryToldInTheLastThreeDaysIsNotToldFirstAgain(t *testing.T) {
	found := findings(t)

	offer := found.Offer(2, []posts.Telling{told(day18, "PALAU WINS DAY 18", 2*time.Hour)}, foundAt)

	assert.Empty(t, offer.First)
	assert.Equal(t, 2, offer.Slots())
}

func TestARunWithNoFrontTourOrPlaceAsksNobody(t *testing.T) {
	found := findings(t)
	found.Fronts, found.Tours, found.Places = nil, nil, nil

	assert.False(t, found.Offer(2, nil, foundAt).Open())
	assert.False(t, findings(t).Offer(1, nil, foundAt).Open(), "the day won fills the run")
}

func TestTheRunIsItsSeasonalStoryFirstThenTheEditorsInItsOrderWithTheirKeys(t *testing.T) {
	found := findings(t)
	offer := found.Offer(3, nil, foundAt)

	choice := found.Edited(offer, posts.Edition{Stories: []posts.Story{battle, india, tour}}, nil, foundAt)

	require.Len(t, choice.Chosen, 3)
	assert.Equal(t, posts.Chosen{Pick: 1, Story: posts.Story{Seasonal: 1}, Key: day18}, choice.Chosen[0])
	assert.Equal(t, posts.Chosen{Pick: 2, Story: with(battle, func(s *posts.Story) { s.Headline = "FRANCE VS TURKEY" }), Key: posts.StoryKey{Flag: "fr", Continent: "Middle East"}}, choice.Chosen[1])
	assert.Equal(t, posts.Chosen{Pick: 3, Story: india, Key: posts.StoryKey{Flag: "dz", Continent: "South Asia"}}, choice.Chosen[2])
	assert.Equal(t, []posts.Pick{1, 2, 3}, choice.Picks())
	assert.Equal(t, []posts.Skipped{{Story: tour, Why: "the run is full"}}, choice.Skipped)
	assert.Equal(t, 3, choice.Room)
}

func TestAStoryThatDoesNotHoldIsSkippedWithWhyAndTheRunGoesOn(t *testing.T) {
	found := findings(t)
	wrong := with(battle, func(s *posts.Story) { s.Rival = "dz" })

	choice := found.Edited(found.Offer(2, nil, foundAt), posts.Edition{Stories: []posts.Story{wrong, india}}, nil, foundAt)

	assert.Equal(t, []posts.Pick{1, 2}, choice.Picks())
	assert.Equal(t, india, choice.Chosen[1].Story)
	require.Len(t, choice.Skipped, 1)
	assert.Equal(t, wrong, choice.Skipped[0].Story)
	assert.Contains(t, choice.Skipped[0].Why, `"fr" and "dz" did not take from each other`)
}

func TestAHeadlineOfTheLastWeekOrOfThisRunIsSkippedAndSoIsAKeyToldTwiceInTheRun(t *testing.T) {
	found := findings(t)
	posted := []posts.Telling{told(posts.StoryKey{Flag: "fr", Continent: "Europe"}, "France vs Turkey ", 6*24*time.Hour)}
	again := with(india, func(s *posts.Story) { s.Headline = "ALGERIA LOSES ALL OF INDIA" })
	middleEast := posts.Story{Front: 1, Kind: posts.Attack, Attacker: "fr", TakenFrom: []string{"tr"}, Continent: "Middle East", Headline: "FRANCE STORMS THE MIDDLE EAST", Teasers: []string{"FRANCE TAKES THE MIDDLE EAST"}}
	sameKey := with(middleEast, func(s *posts.Story) {
		s.Kind, s.Continent, s.Countries, s.Headline = posts.Invasion, "", []string{"sy"}, "FRANCE TAKES SYRIA"
	})

	choice := found.Edited(found.Offer(6, posted, foundAt), posts.Edition{Stories: []posts.Story{battle, india, again, middleEast, sameKey, tour}}, posted, foundAt)

	require.Len(t, choice.Chosen, 4)
	assert.Equal(t, []posts.Story{india, middleEast, tour}, []posts.Story{choice.Chosen[1].Story, choice.Chosen[2].Story, choice.Chosen[3].Story})
	assert.Equal(t, []string{
		`"FRANCE VS TURKEY" was a headline of the last week, or of this run`,
		`"ALGERIA LOSES ALL OF INDIA" was a headline of the last week, or of this run`,
		"fr in Middle East is told already in this run",
	}, whysOf(choice.Skipped))
}

func whysOf(skipped []posts.Skipped) []string {
	whys := make([]string, 0, len(skipped))
	for _, skip := range skipped {
		whys = append(whys, skip.Why)
	}
	return whys
}

func TestASeasonalStoryIsTheRunsToTellNotTheEditors(t *testing.T) {
	found := findings(t)

	choice := found.Edited(found.Offer(2, nil, foundAt), posts.Edition{Stories: []posts.Story{{Seasonal: 1, Headline: "PALAU TAKES THE DAY"}}}, nil, foundAt)

	assert.Equal(t, []posts.Pick{1}, choice.Picks())
	assert.Equal(t, []string{"a seasonal story is the run's to tell first, not the editor's"}, whysOf(choice.Skipped))
}

func TestAHeadlineTheClipCannotDrawIsSkipped(t *testing.T) {
	found := findings(t)
	offer := found.Offer(5, nil, foundAt)

	for name, c := range map[string]struct{ headline, why string }{
		"too long":  {strings.Repeat("A", 61), "is empty or over 60 letters"},
		"blank":     {"   ", `"   " is empty`},
		"a dash":    {"-- FRANCE", `starts with '-'`},
		"an emoji":  {"FRANCE 🔥", `has '🔥'`},
		"quotes":    {`"FRANCE"`, `starts with '"'`},
		"a newline": {"FRANCE\nVS TURKEY", "FRANCE VS TURKEY"},
	} {
		t.Run(name, func(t *testing.T) {
			choice := found.Edited(offer, posts.Edition{Stories: []posts.Story{with(battle, func(s *posts.Story) { s.Headline = c.headline })}}, nil, foundAt)

			if name == "a newline" {
				require.Len(t, choice.Chosen, 2, "a line break is a space")
				assert.Equal(t, c.why, choice.Chosen[1].Story.Headline)
				return
			}
			assert.Equal(t, []posts.Pick{1}, choice.Picks())
			require.Len(t, choice.Skipped, 1)
			assert.Contains(t, choice.Skipped[0].Why, c.why)
		})
	}
}

func TestAHeadlineKeepsAnyScriptItsPunctuationAndOneSpaceBetweenWords(t *testing.T) {
	found := findings(t)
	kept := with(battle, func(s *posts.Story) { s.Headline = "  Polska:  14 punktów straty! Ça va?  " })

	choice := found.Edited(found.Offer(2, nil, foundAt), posts.Edition{Stories: []posts.Story{kept}}, nil, foundAt)

	require.Len(t, choice.Chosen, 2)
	assert.Equal(t, "POLSKA: 14 PUNKTÓW STRATY! ÇA VA?", choice.Chosen[1].Story.Headline)
}

func TestTheTeasersAreTheEditorsSaidAsAHeadlineLeavingOutTheHeadlineAndOnesTooLongForYouTube(t *testing.T) {
	found := findings(t)
	written := with(battle, func(s *posts.Story) {
		s.Teasers = []string{" The war  for Turkey ", "france vs turkey", strings.Repeat("A", 87), "THE WAR FOR TURKEY", "Turkey fights back"}
	})

	choice := found.Edited(found.Offer(2, nil, foundAt), posts.Edition{Stories: []posts.Story{written}}, nil, foundAt)

	require.Len(t, choice.Chosen, 2)
	assert.Equal(t, []string{"THE WAR FOR TURKEY", "TURKEY FIGHTS BACK"}, choice.Chosen[1].Story.Teasers)
}

func TestAStoryWithNoTeaserForTheCaptionIsSkipped(t *testing.T) {
	found := findings(t)
	written := with(battle, func(s *posts.Story) { s.Teasers = []string{"FRANCE VS TURKEY", " "} })

	choice := found.Edited(found.Offer(2, nil, foundAt), posts.Edition{Stories: []posts.Story{written}}, nil, foundAt)

	assert.Equal(t, []posts.Pick{1}, choice.Picks())
	require.Len(t, choice.Skipped, 1)
	assert.Contains(t, choice.Skipped[0].Why, "the caption has no teaser")
}

func TestWithNoEditionTheRunIsItsSeasonalStoryAlone(t *testing.T) {
	found := findings(t)

	choice := found.Edited(found.Offer(2, nil, foundAt), posts.Edition{}, nil, foundAt)

	assert.Equal(t, []posts.Pick{1}, choice.Picks())
	assert.Equal(t, posts.Story{Seasonal: 1}, choice.Chosen[0].Story)
	assert.Empty(t, choice.Skipped)
}
