package clicks

type Outcome uint8

const (
	Unchanged Outcome = iota

	Taken

	Defended
)

func (o Outcome) String() string {
	switch o {
	case Taken:
		return "taken"
	case Defended:
		return "defended"
	default:
		return "unchanged"
	}
}

func OutcomeOf(owner, flag string, defenders int) Outcome {
	switch {
	case owner == flag:
		return Unchanged
	case defenders > 0:
		return Defended
	default:
		return Taken
	}
}

func (o Outcome) OwnerAfter(owner, flag string) string {
	if o == Taken {
		return flag
	}

	return owner
}
