package bonuses

// Rules is how big each charge is: the same for every caller, and it only changes with a deploy. A client
// reads it once, when the page loads.
type Rules struct {
	// Radians of arc: the aiming ring the client draws is the circle the bomb clears.
	BlastRadius float64

	EnclosureMaxTiles int
	// How much each pool holds.
	SpreadClicks int
	Enclosures   int
}
