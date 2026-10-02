package clicks

type Outcome uint8

const (
	// Must stay the zero value: a click the shadow ban dropped reads as unchanged.
	Unchanged Outcome = iota

	Taken

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

type HomeSoilConfig struct {
	Enabled bool
}

type Grounds interface {
	CountryOf(tile uint32) string
}

func NewHomeSoil(config HomeSoilConfig, grounds Grounds) HomeSoil {
	return HomeSoil{enabled: config.Enabled, grounds: grounds}
}

type HomeSoil struct {
	enabled bool
	grounds Grounds
}

func (h HomeSoil) Outcome(tile uint32, owner, flag string) Outcome {
	if !h.enabled {
		return OutcomeOf(owner, "", flag)
	}

	return OutcomeOf(owner, h.grounds.CountryOf(tile), flag)
}

func (h HomeSoil) Enabled() bool {
	return h.enabled
}
