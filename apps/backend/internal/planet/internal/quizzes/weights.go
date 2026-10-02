package quizzes

import (
	"crypto/rand"
	"math"
	"math/big"
)

type Shares interface {
	Share(country string) float64
}

func (b *Bank) subject() string {
	subjects := b.subjects
	if len(subjects) == 0 {
		return ""
	}

	anywhere := len(b.anywhere) > 0

	weights := make([]float64, 0, len(subjects)+1)
	total := 0.0
	for _, subject := range subjects {
		weight := b.weight(subject, len(subjects))
		weights = append(weights, weight)
		total += weight
	}
	if anywhere {
		weights = append(weights, 1)
		total++
	}

	left := fraction() * total
	for i, weight := range weights {
		if left < weight {
			if i == len(subjects) {
				return ""
			}

			return subjects[i]
		}
		left -= weight
	}

	return subjects[len(subjects)-1]
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
