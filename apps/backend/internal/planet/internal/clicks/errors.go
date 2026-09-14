// Package clicks is the tile game itself: what a click is worth, what the map
// looks like, and what changes when somebody takes a tile. It is one of the
// planet module's parts rather than its "domain" — bonuses is the next, and a
// package named for the layer would have had to hold both.
//
// The root holds the vocabulary the whole part is written in — the sentinels and
// the two types that cross between a use case and an adapter. The rules live one
// level down, one package per use case, under usecases/.
package clicks

import "errors"

// Each sentinel names the thing that was wrong, never the HTTP answer it earns.
// "invalid argument" was a status code wearing a domain hat: it told a reader
// nothing a use case could act on, and it meant every caller error in the game
// had to be the same one. A handler maps these onto Connect codes — that
// translation is the edge's, and two of these happen to land on the same code
// without being the same mistake.
var (
	// ErrUnknownCountry is a country code that is not on the ISO list.
	ErrUnknownCountry = errors.New("unknown country code")

	// ErrTileOutOfRange is a tile id outside the map's 1..maxIndex.
	ErrTileOutOfRange = errors.New("tile id out of range")

	// ErrInvalidTileRange is a map read whose span the map cannot answer.
	ErrInvalidTileRange = errors.New("invalid tile range")

	// ErrThrottled is a caller clicking faster than its allowance. It is a rule
	// of the game as played rather than a property of the transport, which is
	// why the throttle is a decorator over the click use case and not an
	// interceptor over the procedure.
	ErrThrottled = errors.New("too many clicks")
)
