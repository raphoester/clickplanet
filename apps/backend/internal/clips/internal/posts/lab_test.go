package posts_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

var (
	farInvasion = posts.Experiment{Name: "far-invasion", Brief: "A flag far from home.", Format: posts.GlobeReplay}
	callToArms  = posts.Experiment{Name: "call-to-arms", Brief: "A flag losing ground, in its language.", Format: posts.GlobeReplay}
	// France far from home in Turkey, written to the far invasion's brief.
	farFrance = with(battle, func(s *posts.Story) {
		s.Headline, s.Hook = "FRANCE STORMS TURKEY", "why is france in turkey?"
	})
)

func TestAnExperimentsNameIsAPlainWordThatIsNotTheControl(t *testing.T) {
	name, err := posts.ExperimentNameOf("far-invasion")
	require.NoError(t, err)
	assert.Equal(t, posts.ExperimentName("far-invasion"), name)

	for _, wrong := range []string{"", "Far", "far invasion", "-far", "control", strings.Repeat("a", 33)} {
		_, err := posts.ExperimentNameOf(wrong)
		assert.ErrorIs(t, err, posts.ErrInvalidExperiment, wrong)
	}
}

func TestALabHoldsEachExperimentOnceWithItsBrief(t *testing.T) {
	_, err := posts.LabOf(farInvasion, farInvasion)
	require.ErrorIs(t, err, posts.ErrInvalidExperiment)

	_, err = posts.LabOf(posts.Experiment{Name: "far-invasion", Brief: "  "})
	require.ErrorIs(t, err, posts.ErrInvalidExperiment)
}

func TestAVoiceTellsAClipOfTheGlobeAndNeverACarousel(t *testing.T) {
	_, err := posts.LabOf(posts.Experiment{Name: "war-report", Brief: "One thread.", Format: posts.GlobeReplay, Narrated: true})
	require.NoError(t, err)

	_, err = posts.LabOf(posts.Experiment{Name: "war-report", Brief: "One thread.", Format: posts.Carousel, Narrated: true})
	require.ErrorIs(t, err, posts.ErrInvalidExperiment)
}

// LoadedDice(k) rolls k modulo the sides of each roll: the slot's 3 sides first, then the experiments'.
func TestTheLabsSlotIsTheRunsFirstPickTwoTimesInThreeElseItsSecondAndItsExperimentIsEven(t *testing.T) {
	lab, err := posts.LabOf(farInvasion, callToArms)
	require.NoError(t, err)

	assert.Equal(t, posts.LabSlot{At: 1, Experiment: farInvasion}, lab.Slot(2, posts.LoadedDice(0)))
	assert.Equal(t, posts.LabSlot{At: 1, Experiment: callToArms}, lab.Slot(2, posts.LoadedDice(1)))
	assert.Equal(t, posts.LabSlot{At: 2, Experiment: farInvasion}, lab.Slot(2, posts.LoadedDice(2)))
	assert.Equal(t, posts.LabSlot{}, lab.Slot(1, posts.LoadedDice(2)), "a second pick of a run of one is none")
	assert.Equal(t, posts.LabSlot{}, lab.Slot(0, posts.LoadedDice(0)), "a full day")
	assert.Equal(t, posts.LabSlot{}, posts.Lab{}.Slot(2, posts.LoadedDice(0)), "no experiment on, no lab")
}

func TestTheLabsSlotMovesPastTheSeasonalStoriesToldFirstOrIsNone(t *testing.T) {
	found := findings(t)

	assert.Equal(t, 2, found.Offer(3, nil, foundAt, posts.LabSlot{At: 1, Experiment: farInvasion}).Lab.At, "the day won is the first pick")
	assert.Equal(t, 2, found.Offer(3, nil, foundAt, posts.LabSlot{At: 2, Experiment: farInvasion}).Lab.At)
	assert.False(t, found.Offer(1, nil, foundAt, posts.LabSlot{At: 1, Experiment: farInvasion}).Lab.On(), "the day won fills the run")
}

func TestTheLabsStoryIsToldAtItsSlotWithItsHookAndItsLanguageAndTheControlFillsTheRest(t *testing.T) {
	found := findings(t)
	offer := found.Offer(3, nil, foundAt, posts.LabSlot{At: 2, Experiment: farInvasion})

	choice := found.Edited(offer, posts.Edition{Stories: []posts.Story{india, tour}, Lab: []posts.Story{farFrance}}, nil, foundAt)

	require.Len(t, choice.Chosen, 3)
	assert.Equal(t, posts.Story{Seasonal: 1}, choice.Chosen[0].Story)
	lab := choice.Chosen[1]
	assert.Equal(t, posts.Pick(2), lab.Pick)
	assert.Equal(t, posts.ExperimentName("far-invasion"), lab.Story.Experiment)
	assert.Equal(t, "WHY IS FRANCE IN TURKEY?", lab.Story.Hook)
	assert.Equal(t, posts.English, lab.Story.Language, "none is English")
	assert.Equal(t, posts.Chosen{Pick: 3, Story: india, Key: posts.StoryKey{Flag: "dz", Continent: "South Asia"}}, choice.Chosen[2])
	assert.True(t, choice.LabTold)
	assert.Equal(t, []posts.Skipped{{Story: tour, Why: "the run is full"}}, choice.Skipped)
}

func TestALabStoryThatDoesNotHoldLeavesItsSlotToTheControl(t *testing.T) {
	found := findings(t)
	offer := found.Offer(3, nil, foundAt, posts.LabSlot{At: 2, Experiment: callToArms})
	undrawable := with(farFrance, func(s *posts.Story) { s.Hook = "¡FRANCIA!" })

	choice := found.Edited(offer, posts.Edition{Stories: []posts.Story{india, tour}, Lab: []posts.Story{undrawable}}, nil, foundAt)

	require.Len(t, choice.Chosen, 3)
	assert.Equal(t, []posts.Story{india, tour}, []posts.Story{choice.Chosen[1].Story, choice.Chosen[2].Story})
	assert.False(t, choice.LabTold)
	require.Len(t, choice.Skipped, 1)
	assert.Contains(t, choice.Skipped[0].Why, "the lab's call-to-arms: ")
	assert.Contains(t, choice.Skipped[0].Why, "the hook")
}

func TestALabStorysLanguageIsATwoOrThreeLetterCode(t *testing.T) {
	found := findings(t)
	offer := found.Offer(2, nil, foundAt, posts.LabSlot{At: 2, Experiment: callToArms})
	polish := with(farFrance, func(s *posts.Story) { s.Language = " PL " })

	choice := found.Edited(offer, posts.Edition{Lab: []posts.Story{polish}}, nil, foundAt)
	require.Len(t, choice.Chosen, 2)
	assert.Equal(t, "pl", choice.Chosen[1].Story.Language)

	spelled := with(farFrance, func(s *posts.Story) { s.Language = "polish" })
	choice = found.Edited(offer, posts.Edition{Lab: []posts.Story{spelled}}, nil, foundAt)
	assert.False(t, choice.LabTold)
}

func TestTheControlSpeaksEnglishAndOpensOnNoHook(t *testing.T) {
	found := findings(t)
	hooked := with(india, func(s *posts.Story) { s.Hook, s.Language = "LOOK AT INDIA", "pl" })

	choice := found.Edited(found.Offer(2, nil, foundAt, posts.LabSlot{}), posts.Edition{Stories: []posts.Story{hooked}}, nil, foundAt)

	require.Len(t, choice.Chosen, 2)
	assert.Equal(t, india, choice.Chosen[1].Story)
}

func TestTheLabsKeyGoesFirstAndAControlStoryOfItIsSkipped(t *testing.T) {
	found := findings(t)
	offer := found.Offer(3, nil, foundAt, posts.LabSlot{At: 3, Experiment: farInvasion})
	again := with(battle, func(s *posts.Story) { s.Headline = "TURKEY FIGHTS FRANCE" })

	choice := found.Edited(offer, posts.Edition{Stories: []posts.Story{again, india}, Lab: []posts.Story{farFrance}}, nil, foundAt)

	require.Len(t, choice.Chosen, 3)
	assert.Equal(t, india, choice.Chosen[1].Story)
	assert.Equal(t, posts.ExperimentName("far-invasion"), choice.Chosen[2].Story.Experiment)
	assert.Equal(t, []string{"fr in Middle East is told already in this run"}, whysOf(choice.Skipped))
}

func TestALabStoryOfARunWithFewerStoriesIsToldRightAfterTheLast(t *testing.T) {
	found := findings(t)
	offer := found.Offer(3, nil, foundAt, posts.LabSlot{At: 3, Experiment: farInvasion})

	choice := found.Edited(offer, posts.Edition{Lab: []posts.Story{farFrance}}, nil, foundAt)

	require.Len(t, choice.Chosen, 2)
	assert.Equal(t, posts.Pick(2), choice.Chosen[1].Pick)
	assert.Equal(t, posts.ExperimentName("far-invasion"), choice.Chosen[1].Story.Experiment)
}

func TestALabStoryIsSkippedWhenTheRunGaveTheLabNoSlotOrHasOneAlready(t *testing.T) {
	found := findings(t)

	none := found.Edited(found.Offer(3, nil, foundAt, posts.LabSlot{}), posts.Edition{Lab: []posts.Story{farFrance}}, nil, foundAt)
	assert.Equal(t, []string{"the run gave the lab no slot"}, whysOf(none.Skipped))

	offer := found.Offer(3, nil, foundAt, posts.LabSlot{At: 2, Experiment: farInvasion})
	twice := found.Edited(offer, posts.Edition{Lab: []posts.Story{farFrance, india}}, nil, foundAt)
	assert.Equal(t, []string{"the lab tells one story a run"}, whysOf(twice.Skipped))
}

func TestAHookThatSaysTheHeadlineIsNoHook(t *testing.T) {
	found := findings(t)
	offer := found.Offer(2, nil, foundAt, posts.LabSlot{At: 2, Experiment: farInvasion})
	echo := with(farFrance, func(s *posts.Story) { s.Hook = s.Headline })

	choice := found.Edited(offer, posts.Edition{Lab: []posts.Story{echo}}, nil, foundAt)

	assert.False(t, choice.LabTold)
}
