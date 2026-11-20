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
	ownLand  = posts.Story{
		List: posts.OwnLand, Hook: "these 5 countries don't own their own land",
		Ranks: []posts.Rank{
			{Code: "fr", Line: "held by palau"},
			{Code: "tr", Line: "held by france"},
			{Code: "in", Line: "held by france"},
			{Code: "de", Line: "held by brazil"},
			{Code: "br", Line: "held by ireland"},
		},
	}
)

func listed(t *testing.T) posts.Findings {
	t.Helper()
	found := findings(t)
	found.Lists = posts.Lists{
		OwnLand: []posts.OwnLandEntry{
			{Country: "fr", Tiles: 9000, OwnShare: 0, Holder: "pw", HolderShare: 0.81},
			{Country: "tr", Tiles: 8000, OwnShare: 0.01, Holder: "fr", HolderShare: 0.6},
			{Country: "in", Tiles: 6300, OwnShare: 0.02, Holder: "fr", HolderShare: 0.97},
			{Country: "de", Tiles: 5000, OwnShare: 0.05, Holder: "br", HolderShare: 0.5},
			{Country: "br", Tiles: 20000, OwnShare: 0.1, Holder: "ie", HolderShare: 0.4},
		},
		MostAttacked:   []posts.AttackedEntry{{Flag: "tr", Lost: 5300, TakenBy: "fr", TakenByTiles: 5300, Country: "tr"}},
		BiggestEmpires: []posts.EmpireEntry{{Flag: "pw", Tiles: 74000, Share: 0.28, Country: "ru"}},
		AttackedSince:  foundAt.Add(-72 * time.Hour),
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

	choice := found.Edited(offer, posts.Edition{Stories: []posts.Story{india}, Lab: []posts.Story{ownLand}}, nil, foundAt)

	require.Len(t, choice.Chosen, 3)
	assert.True(t, choice.LabTold)
	assert.Equal(t, posts.Chosen{
		Pick: 2,
		Story: posts.Story{
			List: posts.OwnLand, Hook: "THESE 5 COUNTRIES DON'T OWN THEIR OWN LAND", Experiment: "carousel", Language: posts.English,
			Ranks: []posts.Rank{
				{Code: "fr", Line: "HELD BY PALAU"},
				{Code: "tr", Line: "HELD BY FRANCE"},
				{Code: "in", Line: "HELD BY FRANCE"},
				{Code: "de", Line: "HELD BY BRAZIL"},
				{Code: "br", Line: "HELD BY IRELAND"},
			},
		},
		Key: posts.StoryKey{Flag: "fr", Continent: posts.TheWorld, Angle: posts.Listed},
	}, choice.Chosen[1])
	assert.Equal(t, india, choice.Chosen[2].Story)
}

func TestACarouselUntrueToItsListLeavesItsSlotToTheControl(t *testing.T) {
	found := listed(t)
	offer := found.Offer(3, nil, foundAt, posts.LabSlot{At: 2, Experiment: carousel})
	swapped := with(ownLand, func(s *posts.Story) {
		s.Ranks = []posts.Rank{s.Ranks[1], s.Ranks[0], s.Ranks[2], s.Ranks[3], s.Ranks[4]}
	})
	firstLine := func(line string) posts.Story {
		ranks := slices.Clone(ownLand.Ranks)
		ranks[0].Line = line
		return with(ownLand, func(s *posts.Story) { s.Ranks = ranks })
	}

	for why, story := range map[string]posts.Story{
		`rank 1 of own-land is "fr", not "tr"`:     swapped,
		"a carousel tells 5 ranks, not 4":          with(ownLand, func(s *posts.Story) { s.Ranks = s.Ranks[:4] }),
		"most-attacked has 1 entries, not 5":       with(ownLand, func(s *posts.Story) { s.List = posts.MostAttacked }),
		"far-from-home has 0 entries, not 5":       with(ownLand, func(s *posts.Story) { s.List = posts.FarFromHome }),
		`no list is named so: "rankings"`:          with(ownLand, func(s *posts.Story) { s.List = "rankings" }),
		`the rank's line "HELD BY PALAU 81%"`:      firstLine("held by palau 81%"),
		`the cover "" is empty`:                    with(ownLand, func(s *posts.Story) { s.Hook = " " }),
		`the cover "` + strings.Repeat("A", 91):    with(ownLand, func(s *posts.Story) { s.Hook = strings.Repeat("a", 91) }),
		`the rank's line "HELD BY #PALAU" has '#'`: firstLine("held by #palau"),
	} {
		choice := found.Edited(offer, posts.Edition{Stories: []posts.Story{india, tour}, Lab: []posts.Story{story}}, nil, foundAt)

		assert.False(t, choice.LabTold, why)
		assert.Equal(t, []posts.Story{india, tour}, []posts.Story{choice.Chosen[1].Story, choice.Chosen[2].Story}, why)
		require.Len(t, choice.Skipped, 1, why)
		assert.Contains(t, choice.Skipped[0].Why, "the lab's carousel: ", why)
		assert.Contains(t, choice.Skipped[0].Why, why)
	}
}

func TestAListWhoseFiveAreTheFiveOfItsLastPostIsNotToldAgain(t *testing.T) {
	found := listed(t)
	offer := found.Offer(2, nil, foundAt, posts.LabSlot{At: 2, Experiment: carousel})
	last := carouselTold(foundAt.Add(-5*time.Hour), posts.OwnLand, "br", "de", "in", "tr", "fr")

	choice := found.Edited(offer, posts.Edition{Lab: []posts.Story{ownLand}}, []posts.Telling{
		carouselTold(foundAt.Add(-30*time.Hour), posts.OwnLand, "fr", "tr", "in", "de", "pw"), last,
	}, foundAt)

	assert.False(t, choice.LabTold)
	assert.Equal(t, []string{"the lab's carousel: the story does not hold: the 5 of own-land are the 5 of its last post, at 2026-10-26T15:29:00Z"},
		whysOf(choice.Skipped), "in any order")
}

func TestAListIsToldAgainWithOtherFiveOrAfterAnotherListHadTheseFive(t *testing.T) {
	found := listed(t)
	offer := found.Offer(2, nil, foundAt, posts.LabSlot{At: 2, Experiment: carousel})

	newer := found.Edited(offer, posts.Edition{Lab: []posts.Story{ownLand}}, []posts.Telling{
		carouselTold(foundAt.Add(-30*time.Hour), posts.OwnLand, "fr", "tr", "in", "de", "br"),
		carouselTold(foundAt.Add(-5*time.Hour), posts.OwnLand, "fr", "tr", "in", "de", "pw"),
	}, foundAt)
	assert.True(t, newer.LabTold, "its last post had another 5")

	other := found.Edited(offer, posts.Edition{Lab: []posts.Story{ownLand}}, []posts.Telling{
		carouselTold(foundAt.Add(-5*time.Hour), posts.BiggestEmpires, "fr", "tr", "in", "de", "br"),
	}, foundAt)
	assert.True(t, other.LabTold, "another list's 5")
}

func TestACarouselIsNoStoryOfTheControlNorOfAClipsExperiment(t *testing.T) {
	found := listed(t)

	control := found.Edited(found.Offer(2, nil, foundAt, posts.LabSlot{}), posts.Edition{Stories: []posts.Story{ownLand}}, nil, foundAt)
	assert.Len(t, control.Chosen, 1)

	clip := found.Edited(found.Offer(2, nil, foundAt, posts.LabSlot{At: 2, Experiment: farInvasion}), posts.Edition{Lab: []posts.Story{ownLand}}, nil, foundAt)
	assert.False(t, clip.LabTold)
}

func TestAnExperimentIsDrawnAsAClipOrACarousel(t *testing.T) {
	_, err := posts.LabOf(farInvasion, carousel)
	require.NoError(t, err)

	_, err = posts.LabOf(posts.Experiment{Name: "slides", Brief: "Photos."})
	require.ErrorIs(t, err, posts.ErrInvalidExperiment)
}
