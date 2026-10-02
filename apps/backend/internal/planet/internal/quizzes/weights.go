package quizzes

import (
	"crypto/rand"
	"math"
	"math/big"
)

// Shares is how much of the map each country holds, 0 to 1. The tile storage already answers this
// for the toll, so the leaderboard the quiz leans on is the same one the click price is read from —
// there is no second ranking to keep in step.
type Shares interface {
	Share(country string) float64
}

// subject picks the country the next question is about, leaning towards the ones that are winning.
//
// The leaderboard is the most interesting thing on the screen, so the quiz should be about it: a
// player watching Bulgaria run away with the planet would rather be asked about Bulgaria than about
// Kiribati. It is a lean and not a rule, because a quiz that only ever asked about the top three
// would be four questions deep by the end of the week.
//
// A country's weight is 1 + Bias × (its share ÷ an even share). So at Bias 1 a country holding
// nothing like its share is asked about about as often as it ever was, one holding an even slice is
// asked about twice as often, and one holding ten times its slice ten times more. The floor of 1 is
// what keeps every country in the draw however badly it is doing, and it is why this reads as
// "more about the leaders" rather than "only about the leaders".
//
// Bias 0 is a flat draw, and a nil Shares is the same: with no map to read, no country is leading.
func (b *Bank) subject() string {
	subjects := b.subjects
	if len(subjects) == 0 {
		return ""
	}

	// The questions about nowhere in particular are one more slot in the draw, weighted as an
	// average country is, so they come up without ever crowding out the map.
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

	// Only float rounding reaches here.
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

	// An even share is one country's worth of a whole map, so this is "how many times its slice
	// does it hold". A country holding the lot is capped, not because the arithmetic breaks but
	// because a planet one country has nearly won should still ask about somewhere else sometimes.
	times := math.Min(share*float64(countries), maxLean)

	return 1 + b.bias*times
}

// maxLean caps how far ahead of an even share a country is counted as being. 30 leaves the leader
// of a real board perhaps fifteen times more likely to be asked about than a country holding
// nothing — plenty — while a planet that has been swept by one country still asks about the rest.
const maxLean = 30

// fraction is a uniform number in [0, 1). A failed read of the system's randomness answers 0, which
// draws the first subject rather than refusing to ask anything.
func fraction() float64 {
	const resolution = 1 << 53

	drawn, err := rand.Int(rand.Reader, big.NewInt(resolution))
	if err != nil {
		return 0
	}

	return float64(drawn.Int64()) / resolution
}
