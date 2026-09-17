package players

import "time"

// Player is what anybody may know about a player with a username: the name, the stats, and when the account
// was made. Never the account id.
type Player struct {
	Name  Name
	Stats Stats
	// CreatedAt is zero when auth no longer knows the account.
	CreatedAt time.Time
	Admin     bool
}
