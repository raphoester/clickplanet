package bonuses

import "slices"

// Unlocked is every kind progression unlocked for an account, beyond the kinds open to everyone: none for a guest.
type Unlocked []Kind

func (u Unlocked) Holds(kind Kind) bool {
	return slices.Contains(u, kind)
}
