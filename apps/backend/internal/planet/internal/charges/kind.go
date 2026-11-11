package charges

// Kind is a charge as the wallet holds it. The offers name theirs on their own (bonuses.Kind), with the same names:
// they draw a kind and pay it, and the wallet decides how much of it fits.
type Kind string

const (
	KindRefill Kind = "refill"

	KindSpreadClicks Kind = "spread_clicks"

	KindBomb          Kind = "bomb"
	KindEncloseClicks Kind = "enclose_clicks"

	KindShields Kind = "shields"
)

var Kinds = []Kind{KindRefill, KindSpreadClicks, KindBomb, KindEncloseClicks, KindShields}
