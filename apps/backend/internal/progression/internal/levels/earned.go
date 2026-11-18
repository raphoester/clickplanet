package levels

// Gain is the XP one source gives.
type Gain struct {
	Source SourceName
	XP     XP
}

// Earned is the XP a record gives, source by source, in the scale's order. What an account gained from one fact to the
// next is two of them apart.
type Earned []Gain

func (s Scale) EarnedOf(record Record) Earned {
	sources := s.Sources()
	earned := make(Earned, 0, len(sources))
	for _, source := range sources {
		earned = append(earned, Gain{Source: source.Name(), XP: source.XP(record)})
	}
	return earned
}

// Since is what each source gives past what it gave before, in this order. A source with nothing more is left out: one
// that gives less (a record read before the one it follows, a stricter scale) gained nothing. A source before does not
// name gave nothing then.
func (e Earned) Since(before Earned) []Gain {
	gains := []Gain{}
	for _, now := range e {
		if was := before.Of(now.Source); now.XP > was {
			gains = append(gains, Gain{Source: now.Source, XP: now.XP - was})
		}
	}
	return gains
}

// Most is the XP each source gives in either, the higher, in this order and then the sources only other names: what is
// known the account earned, whichever was read first.
func (e Earned) Most(other Earned) Earned {
	most := make(Earned, 0, len(e))
	for _, gain := range e {
		most = append(most, Gain{Source: gain.Source, XP: max(gain.XP, other.Of(gain.Source))})
	}
	for _, gain := range other {
		if !e.names(gain.Source) {
			most = append(most, gain)
		}
	}
	return most
}

// Of is the XP the source gives, 0 when it is not named.
func (e Earned) Of(source SourceName) XP {
	for _, gain := range e {
		if gain.Source == source {
			return gain.XP
		}
	}
	return 0
}

func (e Earned) names(source SourceName) bool {
	for _, gain := range e {
		if gain.Source == source {
			return true
		}
	}
	return false
}

// Gained is what the account's open streams hear of a fact counted: the XP it gave, by source, and its level on its bar
// after it.
type Gained struct {
	Gains    []Gain
	Standing Standing
}

// Told is what an account's open streams hear, in the order it came: a level reached (Reached) or XP gained (Gained).
type Told interface {
	told()
}

// Reached is the level an account was raised to, on its bar.
type Reached struct {
	Standing Standing
}

func (Reached) told() {}

func (Gained) told() {}
