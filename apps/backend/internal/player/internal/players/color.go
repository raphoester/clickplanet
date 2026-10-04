package players

import "errors"

// Color is a player.v1.NameColor; zero is no choice, which the client draws grey.
type Color int32

var ErrInvalidColor = errors.New("not a name color")
