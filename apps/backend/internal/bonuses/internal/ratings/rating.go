package ratings

import "math"

// The model (see docs/quiz-difficulty.md): a question has one rating, higher is harder, and a player will have one on
// the same scale; 0 is the average. A player rated s answers a question rated d right with
//
//	P = 1/3 + 2/3 · σ(s − d)
//
// where 1/3 is a lucky guess among three choices: a question nobody knows sits near a third right. Players are not
// rated yet: every answer is an average player's, s = 0.

const (
	// A lucky guess among the three choices a round shows.
	guess = 1.0 / 3

	// MinChance and MaxChance bound a rating, as the chance an average player gets its question right: a question
	// nobody knows (1/3) and one everybody does (1) would be infinitely hard and easy.
	MinChance = 0.34
	MaxChance = 0.99

	// A question's prior is worth this many answers: its first answer weighs 1/11.
	priorAnswers = 10

	// From the 190th answer on, each answer weighs 1/200: the rating follows a question whose wording or whose players
	// changed, rather than averaging over its whole life.
	floorWeight = 1.0 / 200
)

// Every answer is an average player's until players are rated.
const averagePlayer = 0.0

// Rating is how hard a question is: its level, and the answers that moved it there from its prior.
type Rating struct {
	level float64

	answers int
	right   int
}

// PriorOf is a question no answer has moved yet, that this share of average players gets right.
func PriorOf(chance float64) Rating {
	return Rating{level: LevelOf(chance)}
}

// RatingOf is a rating kept: its level, how many answers moved it, and how many of them were right.
func RatingOf(level float64, answers, right int) Rating {
	return Rating{level: level, answers: answers, right: right}
}

// Level is the rating itself: 0 is the average question, higher is harder.
func (r Rating) Level() float64 { return r.level }

// Answers is how many answers moved it.
func (r Rating) Answers() int { return r.answers }

// Right is how many of them were right.
func (r Rating) Right() int { return r.right }

// Chance is the share of average players the rating says get its question right.
func (r Rating) Chance() float64 {
	return chanceOf(averagePlayer, r.level)
}

// Answered is the rating after one more answer by an average player.
//
// The answer moves the chance the rating gave it a step of the way to what happened, 1 for right and 0 for wrong, and
// the level is read back off the moved chance. To first order that is a Fisher-scoring step on the log-likelihood of
// P, Δd = −w·(y − P)/P′, taken exactly on the chance scale, where a step near either end cannot overshoot. With players
// at 0 it is exactly a running mean of the answers on the chance scale, the prior counting as priorAnswers of them,
// until the step reaches its floor: from there each answer weighs the same.
func (r Rating) Answered(correct bool) Rating {
	answers := r.answers + 1
	step := math.Max(1/float64(answers+priorAnswers), floorWeight)

	expected := chanceOf(averagePlayer, r.level)
	outcome, right := 0.0, r.right
	if correct {
		outcome, right = 1, right+1
	}

	return Rating{
		level:   levelOf(averagePlayer, expected+step*(outcome-expected)),
		answers: answers,
		right:   right,
	}
}

// LevelOf is the rating of a question that this share of average players gets right: −logit((3·chance − 1)/2), the
// chance kept between MinChance and MaxChance.
func LevelOf(chance float64) float64 {
	return levelOf(averagePlayer, chance)
}

func chanceOf(skill, level float64) float64 {
	return guess + (1-guess)*sigmoid(skill-level)
}

func levelOf(skill, chance float64) float64 {
	kept := math.Min(math.Max(chance, MinChance), MaxChance)
	if math.IsNaN(chance) {
		kept = MinChance
	}

	return skill - logit((kept-guess)/(1-guess))
}

func sigmoid(x float64) float64 {
	return 1 / (1 + math.Exp(-x))
}

func logit(p float64) float64 {
	return math.Log(p / (1 - p))
}
