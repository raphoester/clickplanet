package players

// Author is who a caller is to the other players: the username its account chose, empty when none, and the
// tag of the address it calls from.
type Author struct {
	Name Name
	Tag  Tag
	// Admin is false for an account with no name.
	Admin bool
}
