package ratings_test

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/ratings"
)

// truth is the chance an average player gets a question of this level right, as the model writes it.
func truth(level float64) float64 {
	return 1.0/3 + 2.0/3/(1+math.Exp(level))
}

// seeded draws the same answers at every run.
func seeded(a, b uint64) *rand.Rand {
	return rand.New(rand.NewPCG(a, b)) //nolint:gosec // a test draw, not a secret.
}

// answeredBy is a rating after n answers by average players who get it right with the chance given.
func answeredBy(rating ratings.Rating, chance float64, n int, draws *rand.Rand) ratings.Rating {
	for range n {
		rating = rating.Answered(draws.Float64() < chance)
	}
	return rating
}

func TestAnAverageQuestionIsAnsweredRightTwoTimesInThree(t *testing.T) {
	assert.InDelta(t, 0, ratings.LevelOf(2.0/3), 1e-12, "σ(0) is a half: a third by luck, and half of the rest")
	assert.InDelta(t, 2.0/3, ratings.PriorOf(2.0/3).Chance(), 1e-12)
}

func TestTheRatingIsTheChanceReadBackwards(t *testing.T) {
	for _, chance := range []float64{0.34, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 0.99} {
		assert.InDelta(t, chance, ratings.PriorOf(chance).Chance(), 1e-12, "%v", chance)
		assert.InDelta(t, chance, truth(ratings.LevelOf(chance)), 1e-12, "%v", chance)
	}

	assert.Greater(t, ratings.LevelOf(0.4), ratings.LevelOf(0.6), "fewer right is harder")
}

func TestAPriorAtEitherEndStaysFinite(t *testing.T) {
	for _, chance := range []float64{0, 0.2, 1.0 / 3, 0.34, 0.99, 1, math.NaN()} {
		level := ratings.LevelOf(chance)
		assert.False(t, math.IsInf(level, 0) || math.IsNaN(level), "%v is rated %v", chance, level)
	}

	assert.InDelta(t, ratings.LevelOf(ratings.MinChance), ratings.LevelOf(1.0/3), 1e-12, "a lucky guess is the hardest kept")
	assert.InDelta(t, ratings.LevelOf(ratings.MaxChance), ratings.LevelOf(1), 1e-12, "a sure answer is the easiest kept")
}

func TestARightAnswerMakesAQuestionEasierAndAWrongOneHarder(t *testing.T) {
	prior := ratings.PriorOf(0.6)

	assert.Less(t, prior.Answered(true).Level(), prior.Level())
	assert.Greater(t, prior.Answered(false).Level(), prior.Level())
	assert.Equal(t, 1, prior.Answered(true).Answers())
	assert.Equal(t, 1, prior.Answered(true).Right())
	assert.Equal(t, 0, prior.Answered(false).Right())
}

// The prior is worth ten answers, and each answer weighs 1/(n + 10): the chance is the running mean of the answers with
// the prior counted ten times, exactly, until the step reaches its floor.
func TestTheChanceIsARunningMeanOfTheAnswersWithThePriorCountedTenTimes(t *testing.T) {
	draws := seeded(7, 7)
	rating := ratings.PriorOf(0.6)

	right := 0
	for n := 1; n <= 190; n++ {
		correct := draws.Float64() < 0.75
		if correct {
			right++
		}
		rating = rating.Answered(correct)

		require.InDelta(t, (10*0.6+float64(right))/float64(10+n), rating.Chance(), 1e-9, "after %d answers", n)
		require.Equal(t, n, rating.Answers())
		require.Equal(t, right, rating.Right())
	}
}

func TestPastTheFloorEachAnswerWeighsOneTwoHundredth(t *testing.T) {
	rating := ratings.RatingOf(ratings.LevelOf(0.5), 1000, 500)

	assert.InDelta(t, 0.5+0.5/200, rating.Answered(true).Chance(), 1e-12)
	assert.InDelta(t, 0.5-0.5/200, rating.Answered(false).Chance(), 1e-12)
}

func TestAnswersNeverPushARatingPastEitherEnd(t *testing.T) {
	sure, hopeless := ratings.PriorOf(0.6), ratings.PriorOf(0.6)
	for range 2000 {
		sure, hopeless = sure.Answered(true), hopeless.Answered(false)
	}

	assert.InDelta(t, ratings.MaxChance, sure.Chance(), 1e-9)
	assert.InDelta(t, ratings.MinChance, hopeless.Chance(), 1e-9)
}

// The simulation the model is held to: questions of known levels, answered by average players, end at their true chance
// whatever their prior. Past the floor a rating follows its last two hundred answers or so, so one question wanders
// about 0.025 around its chance; forty of each level and prior average it out.
func TestSyntheticQuestionsConvergeToTheirTrueChance(t *testing.T) {
	draws := seeded(2026, 11)

	const replicas = 40
	for _, level := range []float64{-3, -2, -1, 0, 1, 2, 3} {
		for _, prior := range []float64{0.4, 0.6, 0.9} {
			mean := 0.0
			for range replicas {
				rating := answeredBy(ratings.PriorOf(prior), truth(level), 3000, draws)
				mean += rating.Chance() / replicas

				require.InDelta(t, truth(level), rating.Chance(), 0.12,
					"a question of level %v, from a prior of %v", level, prior)
			}

			assert.InDelta(t, truth(level), mean, 0.015, "questions of level %v, from a prior of %v", level, prior)
		}
	}
}

func TestAQuestionAnsweredAtChanceLevelEndsThere(t *testing.T) {
	draws := seeded(3, 3)

	rating := answeredBy(ratings.PriorOf(0.9), 1.0/3, 3000, draws)

	assert.GreaterOrEqual(t, rating.Chance(), ratings.MinChance)
	assert.Less(t, rating.Chance(), 0.4, "a question nobody knows is answered by luck alone")
}

// What a question answered about once a week meets: a few hundred answers, and a prior that may be wrong.
func TestABankOfWrongPriorsIsCorrectedByItsAnswers(t *testing.T) {
	draws := seeded(11, 2026)

	const questions = 300
	var priorError, ratedError, early float64
	for range questions {
		level := -3 + 6*draws.Float64()
		prior := 0.4 + 0.5*draws.Float64()

		rating := answeredBy(ratings.PriorOf(prior), truth(level), 30, draws)
		early += math.Abs(rating.Chance() - truth(level))

		rating = answeredBy(rating, truth(level), 370, draws)
		priorError += math.Abs(prior - truth(level))
		ratedError += math.Abs(rating.Chance() - truth(level))
	}

	assert.Less(t, early/questions, priorError/questions/2, "thirty answers already take half the prior's error away")
	assert.Less(t, ratedError/questions, 0.035, "four hundred answers find the true chance")
}
