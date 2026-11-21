package books

import (
	"errors"
	"fmt"
	"time"
)

// Ranking places a round's countries from the map at its end, and says which of them score.
type Ranking interface {
	// Name is the ranking's word in the stores.
	Name() string
	places(start Start, end Snapshot) map[Country]place
}

// place is a country's place in a ranking, compared key by key, the smaller first, and whether it scores there.
type place struct {
	keys   [3]uint32
	scores bool
}

// GainRanking ranks the tiles gained alone, and only a gain scores: holding ground pays nothing, so a peace pays nothing.
type GainRanking struct{}

func (GainRanking) Name() string { return "gain" }

func (GainRanking) places(start Start, end Snapshot) map[Country]place {
	_, gains := holdingsOf(start, end)
	gainRanks := ranksOf(gains)
	places := make(map[Country]place, len(gains))
	for country, gain := range gains {
		places[country] = place{keys: [3]uint32{gainRanks[country]}, scores: gain > 0}
	}
	return places
}

// HeldAndGainRanking ranks the sum of the two ranks, the gain's rank breaking a tie, and every country placed scores:
// the ground held counts as much as the ground gained.
type HeldAndGainRanking struct{}

func (HeldAndGainRanking) Name() string { return "held_and_gain" }

func (HeldAndGainRanking) places(start Start, end Snapshot) map[Country]place {
	held, gains := holdingsOf(start, end)
	heldRanks, gainRanks := ranksOf(held), ranksOf(gains)
	places := make(map[Country]place, len(held))
	for country := range held {
		places[country] = place{keys: [3]uint32{heldRanks[country] + gainRanks[country], gainRanks[country]}, scores: true}
	}
	return places
}

// FightersRanking is a battle's: only the countries whose players took a tile in the round are placed, by the sum of
// their ranks on the tiles they hold at its end and on the tiles their players took in it, the takes' rank breaking a
// tie. Every one placed scores, so a country that held its capital and never fought scores nothing, and one eliminated
// after fighting still scores by the takes it made. A battle counts from nothing, so the ground held is the ground
// won.
type FightersRanking struct{}

func (FightersRanking) Name() string { return "fighters" }

func (FightersRanking) places(_ Start, end Snapshot) map[Country]place {
	sums := fightersOf(end).sums()
	places := make(map[Country]place, len(sums))
	for country, sum := range sums {
		places[country] = place{keys: [3]uint32{sum[0], sum[1]}, scores: true}
	}
	return places
}

// KnockOutRanking places a knock-out battle's fighters: those still standing first, as FightersRanking places them,
// then those knocked out, the last one out first, the more tiles taken breaking a tie of two out at once.
type KnockOutRanking struct{}

func (KnockOutRanking) Name() string { return "knock_out" }

const (
	stillStanding uint32 = iota
	knockedOut
)

func (KnockOutRanking) places(_ Start, end Snapshot) map[Country]place {
	standing, out := fightersOf(end).standingAndOut(end.Out)
	places := make(map[Country]place, len(standing.held)+len(out.held))
	for country, sum := range standing.sums() {
		places[country] = place{keys: [3]uint32{stillStanding, sum[0], sum[1]}, scores: true}
	}
	outAt := make(map[Country]int64, len(out.held))
	for country := range out.held {
		outAt[country] = end.Out[country].UnixNano()
	}
	outRanks, takenRanks := ranksOf(outAt), ranksOf(out.taken)
	for country := range out.held {
		places[country] = place{keys: [3]uint32{knockedOut, outRanks[country], takenRanks[country]}, scores: true}
	}
	return places
}

func holdingsOf(start Start, end Snapshot) (held, gains map[Country]int64) {
	held, gains = map[Country]int64{}, map[Country]int64{}
	for country, tiles := range end.Held {
		if tiles > 0 {
			held[country], gains[country] = int64(tiles), int64(tiles)-int64(start[country])
		}
	}
	return held, gains
}

// fighters is what each country whose players took a tile in the round holds at its end, and the tiles they took.
type fighters struct {
	held, taken map[Country]int64
}

func fightersOf(end Snapshot) fighters {
	all := fighters{held: map[Country]int64{}, taken: map[Country]int64{}}
	for country, taken := range end.Fights {
		if taken > 0 {
			all.held[country], all.taken[country] = int64(end.Held[country]), int64(taken)
		}
	}
	return all
}

func (f fighters) standingAndOut(out map[Country]time.Time) (standing, knocked fighters) {
	standing = fighters{held: map[Country]int64{}, taken: map[Country]int64{}}
	knocked = fighters{held: map[Country]int64{}, taken: map[Country]int64{}}
	for country, held := range f.held {
		group := &standing
		if _, isOut := out[country]; isOut {
			group = &knocked
		}
		group.held[country], group.taken[country] = held, f.taken[country]
	}
	return standing, knocked
}

func (f fighters) sums() map[Country][2]uint32 {
	heldRanks, takenRanks := ranksOf(f.held), ranksOf(f.taken)
	sums := make(map[Country][2]uint32, len(f.held))
	for country := range f.held {
		sums[country] = [2]uint32{heldRanks[country] + takenRanks[country], takenRanks[country]}
	}
	return sums
}

var ErrUnknownRanking = errors.New("no ranking has this name")

func RankingNamed(name string) (Ranking, error) {
	switch name {
	case GainRanking{}.Name():
		return GainRanking{}, nil
	case HeldAndGainRanking{}.Name():
		return HeldAndGainRanking{}, nil
	case FightersRanking{}.Name():
		return FightersRanking{}, nil
	case KnockOutRanking{}.Name():
		return KnockOutRanking{}, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownRanking, name)
	}
}
