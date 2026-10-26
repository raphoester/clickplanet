package posts_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

var (
	length = posts.Experiment{Name: "length", Arms: posts.EvenArms("as-chosen", "short"), Control: "as-chosen"}
	music  = posts.Experiment{
		Name: "music", Arms: []posts.Arm{{Name: "anthem", Weight: 3}, {Name: "trending", Weight: 1}}, Control: "anthem",
	}
	sound = posts.Experiment{
		Name:    "sound",
		Arms:    []posts.Arm{{Name: "anthem", Weight: 1}, {Name: "trending", Weight: 1}, {Name: posts.FellBack}},
		Control: "anthem",
	}
)

func experimentsOf(t *testing.T, experiments ...posts.Experiment) posts.Experiments {
	t.Helper()

	of, err := posts.ExperimentsOf(experiments...)
	require.NoError(t, err)
	return of
}

func TestAnEvenSplitGivesEachArmOneFaceOfTheCoin(t *testing.T) {
	experiments := experimentsOf(t, length)

	assert.Equal(t, posts.Variants{"length": "as-chosen"}, experiments.Assign(posts.LoadedDice(0)))
	assert.Equal(t, posts.Variants{"length": "short"}, experiments.Assign(posts.LoadedDice(1)))
}

func TestAnArmGetsItsWeightOfTheFaces(t *testing.T) {
	experiments := experimentsOf(t, music)

	arms := map[string]int{}
	for face := range 4 {
		arms[experiments.Assign(posts.LoadedDice(face))["music"]]++
	}

	assert.Equal(t, map[string]int{"anthem": 3, "trending": 1}, arms)
}

// Two experiments at once: every pair of arms comes up, so each experiment's effect can be read across the other's.
func TestEachExperimentFlipsACoinOfItsOwnSoTwoFormAFactorialDesign(t *testing.T) {
	experiments := experimentsOf(t, length, music)
	dice := &posts.RolledDice{Faces: []int{0, 0, 1, 0, 0, 3, 1, 3}}

	pairs := make([]posts.Variants, 0, 4)
	for range 4 {
		pairs = append(pairs, experiments.Assign(dice))
	}

	assert.Equal(t, []posts.Variants{
		{"length": "as-chosen", "music": "anthem"},
		{"length": "short", "music": "anthem"},
		{"length": "as-chosen", "music": "trending"},
		{"length": "short", "music": "trending"},
	}, pairs)
}

func TestNoExperimentPutsAClipInNoArm(t *testing.T) {
	assert.Equal(t, posts.Variants{}, experimentsOf(t).Assign(posts.LoadedDice(1)))
}

func TestEachStoryChosenIsPutInItsOwnArmsInTheOrderOfTheRun(t *testing.T) {
	chosen, err := posts.ShortlistOf(
		posts.Story{Title: "A", Key: posts.StoryKey{Flag: "fr", Continent: "Europe"}},
		posts.Story{Title: "B", Key: posts.StoryKey{Flag: "dz", Continent: "Africa"}},
	)
	require.NoError(t, err)

	assigned := experimentsOf(t, length).AssignEach(chosen, map[posts.Pick]string{2: "ALGERIA IS EVERYWHERE"}, &posts.RolledDice{Faces: []int{1, 0}})

	assert.Equal(t, []posts.Assignment{
		{Pick: 1, Variants: posts.Variants{"length": "short"}},
		{Pick: 2, Variants: posts.Variants{"length": "as-chosen"}, Headline: "ALGERIA IS EVERYWHERE"},
	}, assigned)
}

func TestAnArmOfNoWeightIsNeverDrawnButIsAnArmAClipCanBeIn(t *testing.T) {
	experiments := experimentsOf(t, sound)

	drawn := map[string]int{}
	for face := range 2 {
		drawn[experiments.Assign(posts.LoadedDice(face))["sound"]]++
	}

	assert.Equal(t, map[string]int{"anthem": 1, "trending": 1}, drawn)
	require.NoError(t, experiments.Check(posts.Variants{"sound": "fell-back"}))
}

func TestAnExperimentThatCannotRunIsRefused(t *testing.T) {
	for name, experiments := range map[string][]posts.Experiment{
		"no name":           {{Arms: posts.EvenArms("a", "b"), Control: "a"}},
		"two of one name":   {length, length},
		"one arm":           {{Name: "x", Arms: posts.EvenArms("a"), Control: "a"}},
		"an arm twice":      {{Name: "x", Arms: posts.EvenArms("a", "a"), Control: "a"}},
		"an arm of no name": {{Name: "x", Arms: posts.EvenArms("a", ""), Control: "a"}},
		"one arm drawn":     {{Name: "x", Arms: []posts.Arm{{Name: "a", Weight: 1}, {Name: "b"}}, Control: "a"}},
		"a weight under 0":  {{Name: "x", Arms: []posts.Arm{{Name: "a", Weight: 1}, {Name: "b", Weight: 1}, {Name: "c", Weight: -1}}, Control: "a"}},
		"no control":        {{Name: "x", Arms: posts.EvenArms("a", "b")}},
		"a control no arm":  {{Name: "x", Arms: posts.EvenArms("a", "b"), Control: "c"}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := posts.ExperimentsOf(experiments...)

			require.ErrorIs(t, err, posts.ErrInvalidExperiment)
		})
	}
}

func TestAClipInAnArmOfAnExperimentThatRunsIsFine(t *testing.T) {
	experiments := experimentsOf(t, length, music)

	require.NoError(t, experiments.Check(posts.Variants{"length": "short", "music": "trending"}))
	require.NoError(t, experiments.Check(posts.Variants{"length": "as-chosen"}))
	require.NoError(t, experiments.Check(nil), "a clip made by hand is in no experiment")
}

func TestAClipInAnExperimentThatDoesNotRunOrInAnArmItHasNotIsAnError(t *testing.T) {
	experiments := experimentsOf(t, length)

	err := experiments.Check(posts.Variants{"length": "short", "end-card": "podium"})
	require.ErrorIs(t, err, posts.ErrUnknownVariant)
	assert.Contains(t, err.Error(), `"end-card"`)

	err = experiments.Check(posts.Variants{"length": "long"})
	require.ErrorIs(t, err, posts.ErrUnknownVariant)
	assert.Contains(t, err.Error(), `"long"`)
}

func TestArmsAreSaidByExperimentInOrderOfName(t *testing.T) {
	assert.Equal(t, "length short, music anthem", posts.Variants{"music": "anthem", "length": "short"}.String())
	assert.Empty(t, posts.Variants{}.String())
}
