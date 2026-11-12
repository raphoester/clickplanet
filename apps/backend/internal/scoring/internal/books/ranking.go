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
	// fighting is whether only the countries whose players took a tile in the round are placed, on the tiles they took
	// in it instead of the tiles they gained since it opened.
	fighting() bool
}

// place is a country's place in a ranking, compared key by key, the smaller first.
type place [2]uint32

// GainRanking ranks the tiles gained alone, and only a gain scores: holding ground pays nothing, so a peace pays nothing.
type GainRanking struct{}

func (GainRanking) Name() string { return "gain" }

func (GainRanking) place(_, gainRank uint32) place { return place{gainRank} }

func (GainRanking) scores(gain int64) bool { return gain > 0 }

func (GainRanking) fighting() bool { return false }

// HeldAndGainRanking ranks the sum of the two ranks, the gain's rank breaking a tie, and every country placed scores:
// the ground held counts as much as the ground gained.
type HeldAndGainRanking struct{}

func (HeldAndGainRanking) Name() string { return "held_and_gain" }

func (HeldAndGainRanking) place(heldRank, gainRank uint32) place {
	return place{heldRank + gainRank, gainRank}
}

func (HeldAndGainRanking) scores(int64) bool { return true }

func (HeldAndGainRanking) fighting() bool { return false }

// FightersRanking is a battle's: only the countries whose players took a tile in the round are placed, by the sum of
// their ranks on the tiles they hold at its end and on the tiles their players took in it, the takes' rank breaking a
// tie. Every one placed scores, so a country that held its capital and never fought scores nothing, and one eliminated
// after fighting still scores by the takes it made. A battle counts from nothing, so the ground held is the ground
// won.
type FightersRanking struct{}

func (FightersRanking) Name() string { return "fighters" }

func (FightersRanking) place(heldRank, takesRank uint32) place {
	return place{heldRank + takesRank, takesRank}
}

func (FightersRanking) scores(int64) bool { return true }

func (FightersRanking) fighting() bool { return true }

var ErrUnknownRanking = errors.New("no ranking has this name")

func RankingNamed(name string) (Ranking, error) {
	switch name {
	case GainRanking{}.Name():
		return GainRanking{}, nil
	case HeldAndGainRanking{}.Name():
		return HeldAndGainRanking{}, nil
	case FightersRanking{}.Name():
		return FightersRanking{}, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownRanking, name)
	}
}
