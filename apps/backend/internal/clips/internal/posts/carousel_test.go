package posts_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

var (
	carousel = posts.Experiment{Name: "carousel", Brief: "A list of countries.", Format: posts.Carousel}
	fighting = posts.Story{
		List: posts.FightTheMost, Hook: "the 5 countries that fight the most",
		Ranks: []posts.Rank{
			{Code: "fr", Line: "against palau"},
			{Code: "tr", Line: "against france"},
			{Code: "in", Line: "against france"},
			{Code: "de", Line: "against brazil"},
			{Code: "br", Line: "against ireland"},
		},
	}
)

func listed(t *testing.T) posts.Findings {
	t.Helper()
	found := findings(t)
	found.Lists = posts.Lists{
		FightTheMost: []posts.FightEntry{
			{Flag: "fr", Took: 30000, Lost: 11200, Rival: "pw", RivalTiles: 9000, Country: "fr"},
			{Flag: "tr", Took: 9000, Lost: 11000, Rival: "fr", RivalTiles: 8000, Country: "tr"},
			{Flag: "in", Took: 4000, Lost: 5000, Rival: "fr", RivalTiles: 3000, Country: "in"},
			{Flag: "de", Took: 2500, Lost: 2500, Rival: "br", RivalTiles: 2000, Country: "de"},
			{Flag: "br", Took: 2000, Lost: 6000, Rival: "ie", RivalTiles: 900, Country: "br"},
		},
		BiggestEmpires: []posts.EmpireEntry{{Flag: "pw", Tiles: 74000, Share: 0.28, Country: "ru"}},
		FoughtSince:    foundAt.Add(-72 * time.Hour),
	}
	return found
}

func carouselTold(at time.Time, list posts.ListName, ranks ...string) posts.Telling {
	return posts.Telling{
		Key: posts.StoryKey{Flag: ranks[0], Continent: posts.TheWorld, Angle: posts.Listed}, Countries: ranks, At: at,
		Title: "OLD COVER", List: list, Ranks: ranks,
	}
}

func TestACarouselTrueToItsListIsToldAtTheLabsSlotUnderItsCover(t *testing.T) {
	found := listed(t)
	offer := found.Offer(3, nil, foundAt, posts.LabSlot{At: 2, Experiment: carousel})

	choice := found.Edited(offer, posts.Edition{Stories: []posts.Story{india}, Lab: []posts.Story{fighting}}, nil, foundAt)

	require.Len(t, choice.Chosen, 3)
	assert.True(t, choice.LabTold)
	assert.Equal(t, posts.Chosen{
		Pick: 2,
		Story: posts.Story{
			List: posts.FightTheMost, Hook: "THE 5 COUNTRIES THAT FIGHT THE MOST", Experiment: "carousel", Language: posts.English,
			Ranks: []posts.Rank{
				{Code: "fr", Line: "AGAINST PALAU"},
				{Code: "tr", Line: "AGAINST FRANCE"},
				{Code: "in", Line: "AGAINST FRANCE"},
				{Code: "de", Line: "AGAINST BRAZIL"},
				{Code: "br", Line: "AGAINST IRELAND"},
			},
		},
		Key: posts.StoryKey{Flag: "fr", Continent: posts.TheWorld, Angle: posts.Listed},
	}, choice.Chosen[1])
	assert.Equal(t, india, choice.Chosen[2].Story)
}

func TestACarouselUntrueToItsListLeavesItsSlotToTheControl(t *testing.T) {
	found := listed(t)
	offer := found.Offer(3, nil, foundAt, posts.LabSlot{At: 2, Experiment: carousel})
	swapped := with(fighting, func(s *posts.Story) {
		s.Ranks = []posts.Rank{s.Ranks[1], s.Ranks[0], s.Ranks[2], s.Ranks[3], s.Ranks[4]}
	})
	firstLine := func(line string) posts.Story {
		ranks := slices.Clone(fighting.Ranks)
		ranks[0].Line = line
		return with(fighting, func(s *posts.Story) { s.Ranks = ranks })
	}

	for why, story := range map[string]posts.Story{
		`rank 1 of fight-the-most is "fr", not "tr"`: swapped,
		"a carousel tells 5 ranks, not 4":            with(fighting, func(s *posts.Story) { s.Ranks = s.Ranks[:4] }),
		"biggest-empires has 1 entries, not 5":       with(fighting, func(s *posts.Story) { s.List = posts.BiggestEmpires }),
		"far-from-home has 0 entries, not 5":         with(fighting, func(s *posts.Story) { s.List = posts.FarFromHome }),
		`no list is named so: "rankings"`:            with(fighting, func(s *posts.Story) { s.List = "rankings" }),
		`the rank's line "AGAINST PALAU 81%"`:        firstLine("against palau 81%"),
		`the cover "" is empty`:                      with(fighting, func(s *posts.Story) { s.Hook = " " }),
		`the cover "` + strings.Repeat("A", 91):      with(fighting, func(s *posts.Story) { s.Hook = strings.Repeat("a", 91) }),
		`the rank's line "AGAINST #PALAU" has '#'`:   firstLine("against #palau"),
	} {
		choice := found.Edited(offer, posts.Edition{Stories: []posts.Story{india, tour}, Lab: []posts.Story{story}}, nil, foundAt)

		assert.False(t, choice.LabTold, why)
		assert.Equal(t, []posts.Story{india, tour}, []posts.Story{choice.Chosen[1].Story, choice.Chosen[2].Story}, why)
		require.Len(t, choice.Skipped, 1, why)
		assert.Contains(t, choice.Skipped[0].Why, "the lab's carousel: ", why)
		assert.Contains(t, choice.Skipped[0].Why, why)
	}
}

func TestAListWhoseNumbersGrowFromTheFirstIsNotTold(t *testing.T) {
	found := listed(t)
	found.Lists.FarFromHome = []posts.FarLandEntry{
		{Flag: "mx", Country: "my", Tiles: 900, Km: 12300},
		{Flag: "ar", Country: "id", Tiles: 700, Km: 13200},
		{Flag: "gb", Country: "au", Tiles: 800, Km: 10000},
		{Flag: "br", Country: "jp", Tiles: 300, Km: 15600},
		{Flag: "fr", Country: "pe", Tiles: 1200, Km: 9700},
	}
	offer := found.Offer(2, nil, foundAt, posts.LabSlot{At: 2, Experiment: carousel})
	farthest := posts.Story{
		List: posts.FarFromHome, Hook: "which flag fights the farthest from home",
		Ranks: []posts.Rank{
			{Code: "mx", Line: "fighting in malaysia"},
			{Code: "ar", Line: "fighting in indonesia"},
			{Code: "gb", Line: "fighting in australia"},
			{Code: "br", Line: "fighting in japan"},
			{Code: "fr", Line: "fighting in peru"},
		},
	}

	choice := found.Edited(offer, posts.Edition{Lab: []posts.Story{farthest}}, nil, foundAt)

	assert.False(t, choice.LabTold)
	assert.Equal(t, []string{"the lab's carousel: the story does not hold: rank 2 of far-from-home shows more than rank 1: 13200 against 12300"},
		whysOf(choice.Skipped))
}

func TestAFightListIsRankedByTheTilesTakenAloneNotTheTilesLost(t *testing.T) {
	found := listed(t)
	found.Lists.FightTheMost[0], found.Lists.FightTheMost[1] = found.Lists.FightTheMost[1], found.Lists.FightTheMost[0]
	found.Lists.FightTheMost[0].Lost = 40000
	offer := found.Offer(2, nil, foundAt, posts.LabSlot{At: 2, Experiment: carousel})
	swapped := with(fighting, func(s *posts.Story) {
		s.Ranks = []posts.Rank{s.Ranks[1], s.Ranks[0], s.Ranks[2], s.Ranks[3], s.Ranks[4]}
	})

	choice := found.Edited(offer, posts.Edition{Lab: []posts.Story{swapped}}, nil, foundAt)

	assert.False(t, choice.LabTold)
	assert.Equal(t, []string{"the lab's carousel: the story does not hold: rank 2 of fight-the-most shows more than rank 1: 30000 against 9000"},
		whysOf(choice.Skipped))
}

func TestAListWhoseFiveAreTheFiveOfItsLastPostIsNotToldAgain(t *testing.T) {
	found := listed(t)
	offer := found.Offer(2, nil, foundAt, posts.LabSlot{At: 2, Experiment: carousel})
	last := carouselTold(foundAt.Add(-5*time.Hour), posts.FightTheMost, "br", "de", "in", "tr", "fr")

	choice := found.Edited(offer, posts.Edition{Lab: []posts.Story{fighting}}, []posts.Telling{
		carouselTold(foundAt.Add(-30*time.Hour), posts.FightTheMost, "fr", "tr", "in", "de", "pw"), last,
	}, foundAt)

	assert.False(t, choice.LabTold)
	assert.Equal(t, []string{"the lab's carousel: the story does not hold: the 5 of fight-the-most are the 5 of its last post, at 2026-10-26T15:29:00Z"},
		whysOf(choice.Skipped), "in any order")
}

func TestAListIsToldAgainWithOtherFiveOrAfterAnotherListHadTheseFive(t *testing.T) {
	found := listed(t)
	offer := found.Offer(2, nil, foundAt, posts.LabSlot{At: 2, Experiment: carousel})

	newer := found.Edited(offer, posts.Edition{Lab: []posts.Story{fighting}}, []posts.Telling{
		carouselTold(foundAt.Add(-30*time.Hour), posts.FightTheMost, "fr", "tr", "in", "de", "br"),
		carouselTold(foundAt.Add(-5*time.Hour), posts.FightTheMost, "fr", "tr", "in", "de", "pw"),
	}, foundAt)
	assert.True(t, newer.LabTold, "its last post had another 5")

	other := found.Edited(offer, posts.Edition{Lab: []posts.Story{fighting}}, []posts.Telling{
		carouselTold(foundAt.Add(-5*time.Hour), posts.BiggestEmpires, "fr", "tr", "in", "de", "br"),
	}, foundAt)
	assert.True(t, other.LabTold, "another list's 5")
}

func TestACarouselIsNoStoryOfTheControlNorOfAClipsExperiment(t *testing.T) {
	found := listed(t)

	control := found.Edited(found.Offer(2, nil, foundAt, posts.LabSlot{}), posts.Edition{Stories: []posts.Story{fighting}}, nil, foundAt)
	assert.Len(t, control.Chosen, 1)

	clip := found.Edited(found.Offer(2, nil, foundAt, posts.LabSlot{At: 2, Experiment: farInvasion}), posts.Edition{Lab: []posts.Story{fighting}}, nil, foundAt)
	assert.False(t, clip.LabTold)
}

func TestAnExperimentIsDrawnAsAClipOrACarousel(t *testing.T) {
	_, err := posts.LabOf(farInvasion, carousel)
	require.NoError(t, err)

	_, err = posts.LabOf(posts.Experiment{Name: "slides", Brief: "Photos."})
	require.ErrorIs(t, err, posts.ErrInvalidExperiment)
}
