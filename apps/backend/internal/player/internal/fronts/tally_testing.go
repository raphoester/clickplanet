//go:build testing

package fronts

type Tally struct {
	playsFor     map[Country]uint64
	playsAgainst map[Country]uint64
}

func TallyOf(playsFor, playsAgainst map[Country]uint64) Tally {
	return Tally{playsFor: counted(playsFor), playsAgainst: counted(playsAgainst)}
}

func (t Tally) WithTake(take Take) Tally {
	next := TallyOf(t.playsFor, t.playsAgainst)
	next.playsFor[take.Country()]++
	if against, ok := take.Against(); ok {
		next.playsAgainst[against]++
	}
	return next
}

func (t Tally) PlaysFor() map[Country]uint64 { return counted(t.playsFor) }

func (t Tally) PlaysAgainst() map[Country]uint64 { return counted(t.playsAgainst) }

func counted(tiles map[Country]uint64) map[Country]uint64 {
	kept := make(map[Country]uint64, len(tiles))
	for country, n := range tiles {
		if n > 0 {
			kept[country] = n
		}
	}
	return kept
}
