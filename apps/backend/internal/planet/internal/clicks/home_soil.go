package clicks

// Outcome is what one click does to the tile it lands on. Every click costs one token whatever it is:
// the rule changes what a click does, never what it costs.
type Outcome uint8

const (
	// Unchanged is a tile that already wears the flag clicked. Nothing is written. It is the zero value, so a
	// click that never reached the rule — dropped by the shadow ban — reads as having changed nothing.
	Unchanged Outcome = iota

	// Taken is a tile that wears the flag clicked once the click is done.
	Taken

	// Cleared is a tile on a country's own ground that wore that country's flag, clicked for another flag: it
	// belongs to nobody once the click is done. The next click on it takes it, for any flag.
	Cleared
)

func (o Outcome) String() string {
	switch o {
	case Taken:
		return "taken"
	case Cleared:
		return "cleared"
	default:
		return "unchanged"
	}
}

// OutcomeOf is the home-soil rule: native land takes two clicks. owner holds the tile, empty for nobody;
// ground is the country whose real-world soil the tile is on, empty for none; flag is the country clicked for.
//
// A tile a country holds on its own ground is only cleared by another flag, and the next click takes it.
// Anything else is taken in one click, as it always was: foreign ground, an empty tile, and a native tile held
// by another flag, which its natives win back in one.
func OutcomeOf(owner, ground, flag string) Outcome {
	switch {
	case owner == flag:
		return Unchanged
	case ground != "" && owner == ground:
		return Cleared
	default:
		return Taken
	}
}

// OwnerAfter is who holds the tile once a click for flag had this outcome on a tile owner held.
func (o Outcome) OwnerAfter(owner, flag string) string {
	switch o {
	case Taken:
		return flag
	case Cleared:
		return ""
	default:
		return owner
	}
}

// HomeSoilConfig is the `homeSoil:` block. Off, every ground reads as nobody's home and every click takes.
type HomeSoilConfig struct {
	Enabled bool
}

// Grounds says which country's ground a tile sits on. Borders is one.
type Grounds interface {
	CountryOf(tile uint32) string
}

func NewHomeSoil(config HomeSoilConfig, grounds Grounds) HomeSoil {
	return HomeSoil{enabled: config.Enabled, grounds: grounds}
}

// HomeSoil is the rule as this process plays it: OutcomeOf on the tile's own ground, or on no ground when it
// is switched off. Only the player's click path reads it — the click, the spread and the enclose. The bomb and
// the operator tools write the map without it.
type HomeSoil struct {
	enabled bool
	grounds Grounds
}

// Outcome is what a click for flag does to tile, which owner holds.
func (h HomeSoil) Outcome(tile uint32, owner, flag string) Outcome {
	if !h.enabled {
		return OutcomeOf(owner, "", flag)
	}

	return OutcomeOf(owner, h.grounds.CountryOf(tile), flag)
}

// Enabled says whether native land takes two clicks. The client is told, so it paints its own click right.
func (h HomeSoil) Enabled() bool {
	return h.enabled
}
