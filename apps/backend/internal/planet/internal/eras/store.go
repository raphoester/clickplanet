package eras

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Store keeps the boards planet knows and each map's eras.
type Store interface {
	// Boards are the boards planet keeps a row of.
	Boards(ctx context.Context) ([]Board, error)
	// Latest is the era a map plays: ErrNoEra when it has none.
	Latest(ctx context.Context, played clicks.MapID) (Era, error)
	// Covering are the eras of a map that played some of a window, oldest first: the one it played at since, if any,
	// and each that started after since and before until.
	Covering(ctx context.Context, played clicks.MapID, since, until time.Time) ([]Era, error)
	// Start keeps a map's era on a board planet keeps: one kept already at that time changes nothing.
	Start(ctx context.Context, played clicks.Stage, at time.Time) error
}

// Playing is the stage a map plays, once the boot may load it: its latest era's board is one this process carries.
func Playing(ctx context.Context, store Store, boards Boards, played clicks.MapID) (clicks.Stage, error) {
	era, err := store.Latest(ctx, played)
	if err != nil {
		return clicks.Stage{}, fmt.Errorf("failed to read the era of map %s: %w", played, err)
	}
	return boards.Stage(era)
}

// Maps names the stage each map plays, from its eras and the boards this process carries.
type Maps struct {
	store  Store
	boards Boards
}

func NewMaps(store Store, boards Boards) Maps {
	return Maps{store: store, boards: boards}
}

// Playing is the stage a map plays: its latest era's board, which this process carries.
func (m Maps) Playing(ctx context.Context, played clicks.MapID) (clicks.Stage, error) {
	return Playing(ctx, m.store, m.boards, played)
}
