package domain

import "errors"

// ErrInvalidArgument marks a failure caused by the caller, so that a transport
// can answer 400 rather than 500.
var ErrInvalidArgument = errors.New("invalid argument")
