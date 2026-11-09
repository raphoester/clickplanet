package arenas

import "context"

// Store keeps each arena: when it opened and for which end, and when it closed. Its map is the tile map's, named by its
// id.
type Store interface {
	// Find is the arena kept under id: one never kept is an Arena of that id that never opened.
	Find(ctx context.Context, id ID) (Arena, error)
	// Playing is the arena open now: opened and not closed; none is an Arena that never opened.
	Playing(ctx context.Context) (Arena, error)
	Open(ctx context.Context, opening Opening) error
	Close(ctx context.Context, closing Closing) error
}
