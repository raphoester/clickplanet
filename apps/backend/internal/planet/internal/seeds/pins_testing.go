//go:build testing

package seeds

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// FixedPins are the same flags on a pin at every read.
type FixedPins []clicks.Pinned

func (f FixedPins) Pinned(context.Context) ([]clicks.Pinned, error) {
	return f, nil
}
