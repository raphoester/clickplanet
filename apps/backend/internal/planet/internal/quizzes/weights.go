package quizzes

import (
	"crypto/rand"
	"math"
	"math/big"
)

type Shares interface {
	Share(country string) float64
}

// A question that does not name its subject is answered by it, so the lean would give the answer away.
func (b *Bank) pool() []Question {
	pools := make([][]Question, 0, 2*len(b.subjects)+1)
	weights := make([]float64, 0, cap(pools))

	for _, subject := range b.subjects {
		questions := b.bySubject[subject]
		size := float64(questions.size())

		pools = append(pools, questions.named, questions.unnamed)
		weights = append(weights,
			b.weight(subject, len(b.subjects))*float64(len(questions.named))/size,
			float64(len(questions.unnamed))/size)
	}
	if len(b.anywhere) > 0 {
		pools = append(pools, b.anywhere)
		weights = append(weights, 1)
	}
	if len(pools) == 0 {
		return nil
	}

	return pools[drawn(weights)]
}

func drawn(weights []float64) int {
	total := 0.0
	for _, weight := range weights {
		total += weight
	}

	left := fraction() * total
	for i, weight := range weights {
		if left < weight {
			return i
		}
		left -= weight
	}

	return len(weights) - 1
}

func (b *Bank) weight(subject string, countries int) float64 {
	if b.shares == nil || b.bias <= 0 || countries == 0 {
		return 1
	}

	share := b.shares.Share(subject)
	if math.IsNaN(share) || share <= 0 {
		return 1
	}

	times := math.Min(share*float64(countries), maxLean)

	return 1 + b.bias*times
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
