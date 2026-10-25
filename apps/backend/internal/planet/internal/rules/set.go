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
}
