package bonuses

import (
	"fmt"
	"math"
	"slices"
)

// KindBand is the weights a box is drawn with from Share of the map up: the Mario Kart rule.
type KindBand struct {
	Share float64
	Kinds map[Kind]float64
}

// Bands is bonus.kinds from 0, then each of bonus.kindsByShare from its share.
type Bands []KindBand

// Bands is the table a box's kind is drawn from, defaults filled in.
func (c Config) Bands() Bands {
	c = c.withDefaults()

	return append(Bands{{Share: 0, Kinds: c.Kinds}}, c.KindsByShare...)
}

// At is the last band that starts at or below share.
func (b Bands) At(share float64) KindBand {
	band := b[0]
	for _, next := range b[1:] {
		if share < next.Share {
			break
		}
		band = next
	}

	return band
}

// bandsError refuses shares that do not rise inside (0, 1), and a band that could never draw anything.
func (c Config) bandsError() error {
	previous := 0.0

	for i, band := range c.KindsByShare {
		if math.IsNaN(band.Share) || band.Share <= previous || band.Share >= 1 {
			return fmt.Errorf("bonus.kindsByShare[%d].share is %v: shares must rise, above 0 and below 1", i, band.Share)
		}
		if err := weightsError(fmt.Sprintf("bonus.kindsByShare[%d].kinds", i), band.Kinds); err != nil {
			return err
		}
		previous = band.Share
	}

	return nil
}

// weightsError refuses an unknown kind, a weight that is not a number of 0 or more, and a table with no chance.
func weightsError(name string, kinds map[Kind]float64) error {
	total := 0.0

	for kind, weight := range kinds {
		if !slices.Contains(Kinds, kind) {
			return fmt.Errorf("%s holds %q, which is not one of %v", name, kind, Kinds)
		}
		if weight < 0 || math.IsNaN(weight) || math.IsInf(weight, 0) {
			return fmt.Errorf("%s.%s is %v: a weight must be a number of 0 or more", name, kind, weight)
		}
		total += weight
	}

	if total == 0 {
		return fmt.Errorf("%s gives no kind a weight above 0, so no box could be anything", name)
	}

	return nil
}
