package arenas

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

// Store keeps each arena, and the map from before it while it is open.
type Store interface {
	// Find is the arena kept under id: one never kept is an Arena of that id that never opened.
	Find(ctx context.Context, id ID) (Arena, error)
	// Playing is the arena open now: opened and not closed; none is an Arena that never opened.
	Playing(ctx context.Context) (Arena, error)
	// Open keeps the arena open, and the map from before it.
	Open(ctx context.Context, opening Opening, before clicks.MapCopy) error
	// Close keeps the arena closed and answers the map kept when it opened, which it forgets; an empty one for an arena
	// that never opened.
	Close(ctx context.Context, closing Closing) (clicks.MapCopy, error)
	// ForgetHolder takes a deleted account off every tile an arena keeps, as the map forgets it.
	ForgetHolder(ctx context.Context, account cpsession.AccountID) error
}
