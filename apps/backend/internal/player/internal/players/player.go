package players

import "time"

type Player struct {
	Name      Name
	Stats     Stats
	CreatedAt time.Time
	Admin     bool
}
