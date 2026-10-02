package clicks

import "errors"

var (
	ErrUnknownCountry = errors.New("unknown country code")

	ErrTileOutOfRange = errors.New("tile id out of range")

	ErrBonusesTogether = errors.New("spread and enclose switched on together")

	ErrInvalidTileRange = errors.New("invalid tile range")

	ErrThrottled = errors.New("too many clicks")
)
