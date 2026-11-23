package charges

// Kind is a charge as the wallet holds it; planet.v1.BonusKind is its word on the wire, which whoever grants speaks:
// they draw a kind and pay it, and the wallet decides how much of it fits.
type Kind string

const (
	KindRefill Kind = "refill"

	KindSpreadClicks Kind = "spread_clicks"

	KindBomb          Kind = "bomb"
	KindEncloseClicks Kind = "enclose_clicks"

	KindShields Kind = "shields"

	KindVirus Kind = "virus"

	KindMine Kind = "mine"

	KindRally Kind = "rally"

	KindTrap Kind = "trap"
)

var Kinds = []Kind{
	KindRefill, KindSpreadClicks, KindBomb, KindEncloseClicks, KindShields, KindVirus, KindMine, KindRally, KindTrap,
}
