package clicks

type Outcome uint8

const (
	Unchanged Outcome = iota

	Taken

	Shielded
)

func (o Outcome) String() string {
	switch o {
	case Taken:
		return "taken"
	case Shielded:
		return "shielded"
	default:
		return "unchanged"
	}
}

func OutcomeOf(owner, flag string, shields int) Outcome {
	switch {
	case owner == flag:
		return Unchanged
	case shields > 0:
		return Shielded
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
