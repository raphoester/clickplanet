package bank

import (
	"math"

	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/ratings"
)

// Fit is how the draw weighs a question for how hard it is, inside the pool the lean picked (quizzes.v1.Rules).
type Fit struct {
	// Target is the chance of a right answer, for an average player, the draw aims at.
	Target float64

	// Width is how wide the bell around Target is, on the rating scale. 0 is off: the draw of before, with no floor and
	// no exploring either.
	Width float64

	// A question rated under Floor once FloorAfter answers moved it is never drawn. A prior alone never leaves one out.
	Floor      float64
	FloorAfter int

	// Explore is the share of draws that ignore how hard a question is; they never draw one under the floor either.
	Explore float64
}

func (f Fit) on() bool { return f.Width > 0 }

// below is a question its answers measured under the floor: a prior alone is never measured.
func (f Fit) below(rating ratings.Rating) bool {
	return rating.Answers() >= f.FloorAfter && rating.Answers() > 0 && rating.Chance() < f.Floor
}

// weight is the bell around the target, on the rating scale: 1 on it, 0.61 a width away, 0.14 two widths away.
func (f Fit) weight(rating ratings.Rating) float64 {
	off := (rating.Level() - ratings.LevelOf(f.Target)) / f.Width
	return math.Exp(-off * off / 2)
}

// fitted draws a question of the pool by the bell, or flat on an exploring draw, never one under the floor. False when
// every question of the pool is under it.
func (b *Bank) fitted(pool []Question, fit Fit) (Question, bool) {
	drawable := make([]Question, 0, len(pool))
	weights := make([]float64, 0, len(pool))
	for _, question := range pool {
		rating := b.Rating(question.ID)
		if fit.below(rating) {
			continue
		}
		drawable = append(drawable, question)
		weights = append(weights, fit.weight(rating))
	}
	if len(drawable) == 0 {
		return Question{}, false
	}

	if fraction() < fit.Explore {
		return drawable[index(len(drawable))], true
	}
	if i, ok := drawn(weights); ok {
		return drawable[i], true
	}
	return drawable[index(len(drawable))], true
}
