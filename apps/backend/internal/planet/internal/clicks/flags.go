package clicks

import (
	"context"
	"fmt"
)

// Flags is the catalog of flags (cpflags.Catalog): the ones the game ships with, and the ones opened since.
type Flags interface {
	// A flag closed is known and not playable.
	Playable(ctx context.Context, flag string) (bool, error)
}

// CheckFlag refuses a flag no player may play for: one never opened, or one closed.
func CheckFlag(ctx context.Context, flags Flags, flag string) error {
	playable, err := flags.Playable(ctx, flag)
	if err != nil {
		return fmt.Errorf("failed to read whether %q may be played: %w", flag, err)
	}
	if !playable {
		return fmt.Errorf("%w: %q", ErrUnknownCountry, flag)
	}
	return nil
}
