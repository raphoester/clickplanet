package players

import "errors"

// Color is a player.v1.NameColor; zero is no choice, and the client derives one from the name.
type Color int32

var ErrInvalidColor = errors.New("not a name color")
