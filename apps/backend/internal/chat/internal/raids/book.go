package raids

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Reader interface {
	Current(ctx context.Context) (RaidMode, error)
}

const readTimeout = 5 * time.Second

func NewBook(reader Reader, clock cptime.Clock) Book {
	return Book{reader: reader, clock: clock}
}

type Book struct {
	reader Reader
	clock  cptime.Clock
}

// The raid mode that runs now, if one does.
func (b Book) Running(ctx context.Context) (RaidMode, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	mode, err := b.reader.Current(ctx)
	switch {
	case errors.Is(err, ErrNoRaidMode):
		return RaidMode{}, false, nil
	case err != nil:
		return RaidMode{}, false, fmt.Errorf("failed to read whether raid mode runs: %w", err)
	}
	if !mode.RunsAt(b.clock.Now()) {
		return RaidMode{}, false, nil
	}
	return mode, true, nil
}
