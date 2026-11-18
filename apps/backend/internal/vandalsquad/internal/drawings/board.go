package drawings

import "errors"

// ErrOtherBoard is tiles asked of one board answered on another: the map in play plays another board than the world's,
// an arena of its own.
var ErrOtherBoard = errors.New("planet answered the tiles of another board")

// Board names the board a tile id is a place on, as planet names it: "earth-1". A tile id of another board is another
// place.
type Board string
