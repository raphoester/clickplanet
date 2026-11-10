package books

import (
	"errors"
	"fmt"
)

// Ranking orders the countries that hold tiles at a round's end, from each one's rank on the tiles it holds then and its
// rank on the tiles it gained since the round opened, and says which of them score.
type Ranking interface {
	// Name is the ranking's word in the stores.
	Name() string
	place(heldRank, gainRank uint32) place
	scores(gain int64) bool
}

// place is a country's place in a ranking, compared key by key, the smaller first.
type place [2]uint32

// GainRanking ranks the tiles gained alone, and only a gain scores: holding ground pays nothing, so a peace pays nothing.
type GainRanking struct{}

func (GainRanking) Name() string { return "gain" }

func (GainRanking) place(_, gainRank uint32) place { return place{gainRank} }

func (GainRanking) scores(gain int64) bool { return gain > 0 }

// HeldAndGainRanking ranks the sum of the two ranks, the gain's rank breaking a tie, and every country placed scores:
// the ground held counts as much as the ground gained.
type HeldAndGainRanking struct{}

func (HeldAndGainRanking) Name() string { return "held_and_gain" }

func (HeldAndGainRanking) place(heldRank, gainRank uint32) place {
	return place{heldRank + gainRank, gainRank}
}

func (HeldAndGainRanking) scores(int64) bool { return true }

var ErrUnknownRanking = errors.New("no ranking has this name")

func RankingNamed(name string) (Ranking, error) {
	switch name {
	case GainRanking{}.Name():
		return GainRanking{}, nil
	case HeldAndGainRanking{}.Name():
		return HeldAndGainRanking{}, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownRanking, name)
	}
}
