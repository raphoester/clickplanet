package moments

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

// Kept is what the gamemaster keeps of its calls to scoring: each round it opened, and what scoring answered to each
// close; and each season whose map planet reset, by its book.
type Kept struct {
	Opened       []RoundID
	Scored       []RoundID
	MissedRounds []RoundID
	Closed       []BookID
	MissedBooks  []BookID
	Reset        []BookID
}

// ArenasKept is what the gamemaster keeps of the arena battles: each battle judged, and each arena planet opened and
// closed.
type ArenasKept struct {
	Judged []Judgement
	Opened []RoundID
	Closed []RoundID
}

// Known is what the gamemaster knows of what it commanded: each battle judged, each round and book scoring answered,
// and each arena planet opened and closed. The zero Known knows none: a plan with it opens every round on time and waits for
// nothing, as a read of the plan needs.
type Known struct {
	modes map[RoundID]Mode
	// settled are the rounds scoring scored or missed.
	settled *cpcolls.Set[RoundID]
	// books are the books scoring closed or missed.
	books  *cpcolls.Set[BookID]
	reset  *cpcolls.Set[BookID]
	opened *cpcolls.Set[RoundID]
	closed *cpcolls.Set[RoundID]
}

func NewKnown(kept Kept, arenas ArenasKept) Known {
	known := Known{
		modes:   map[RoundID]Mode{},
		settled: cpcolls.NewSet(append(append([]RoundID{}, kept.Scored...), kept.MissedRounds...)...),
		books:   cpcolls.NewSet(append(append([]BookID{}, kept.Closed...), kept.MissedBooks...)...),
		reset:   cpcolls.NewSet(kept.Reset...),
		opened:  cpcolls.NewSet(arenas.Opened...),
		closed:  cpcolls.NewSet(arenas.Closed...),
	}
	for _, judgement := range arenas.Judged {
		known.modes[judgement.Round] = judgement.Mode
	}
	return known
}

// With is the plan that knows what was commanded: a battle's round opens once it is judged, an arena battle's once its
// arena opened, what comes after an arena battle waits for its arena to close, and a close is due until scoring or
// planet answered it.
func (p Plan) With(known Known) Plan {
	p.known = known
	return p
}

// settled is true once scoring answered a round's close, scored or missed, or its book's: nothing scores it any more.
func (p Plan) settled(book BookID, round RoundID) bool {
	return p.known.books.Contains(book) || p.known.settled.Contains(round)
}
