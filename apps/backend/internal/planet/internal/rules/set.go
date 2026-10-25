package rules

import (
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
)

type SetName string

const Default SetName = "default"

var ErrUnknownSet = errors.New("no rule set has this name")

type Set struct {
	Name SetName

	Tempo float64

	Boxes bonuses.Pace

	AntiBot AntiBot

	Frozen bool
}

type AntiBot struct {
	// Crowd is how many networks of one prefix, in step, are banned as a crowd.
	Crowd int
}
