package quizzes

import (
	"crypto/rand"
	"math"
	"math/big"
)

type Shares interface {
	Share(country string) float64
}

// pools is each subject's two pools, with what the lean weighs each. A question that does not name its subject is
// answered by it, so the lean would give the answer away: only the named pool leans.
func (b *Bank) pools(lean float64) ([][]Question, []float64) {
	pools := make([][]Question, 0, 2*len(b.subjects)+1)
	weights := make([]float64, 0, cap(pools))

	for _, subject := range b.subjects {
		questions := b.bySubject[subject]
		size := float64(questions.size())

		pools = append(pools, questions.named, questions.unnamed)
		weights = append(weights,
			b.weight(subject, len(b.subjects), lean)*float64(len(questions.named))/size,
			float64(len(questions.unnamed))/size)
	}
	if len(b.anywhere) > 0 {
		pools = append(pools, b.anywhere)
		weights = append(weights, 1)
	}

	return pools, weights
}

// drawn picks an index with a chance of its weight over their sum, and false when nothing weighs anything.
func drawn(weights []float64) (int, bool) {
	total := 0.0
	for _, weight := range weights {
		total += weight
	}
	if total <= 0 {
		return 0, false
	}

	left := fraction() * total
	for i, weight := range weights {
		if left < weight {
			return i, true
		}
		left -= weight
	}

	for i := len(weights) - 1; i >= 0; i-- {
		if weights[i] > 0 {
			return i, true
		}
	}
	return 0, false
}

func (b *Bank) weight(subject string, countries int, lean float64) float64 {
	if b.shares == nil || lean <= 0 || countries == 0 {
		return 1
	}

	share := b.shares.Share(subject)
	if math.IsNaN(share) || share <= 0 {
		return 1
	}

	times := math.Min(share*float64(countries), maxLean)

	return 1 + lean*times
}

const maxLean = 30

func fraction() float64 {
	const resolution = 1 << 53

	drawn, err := rand.Int(rand.Reader, big.NewInt(resolution))
	if err != nil {
		return 0
	}

	return float64(drawn.Int64()) / resolution
}
