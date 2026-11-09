package list_due_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// Promo's battles to come.
type Battles interface {
	// Those starting within the window, by start, with their cards.
	Teasers(ctx context.Context, within time.Duration) ([]teasers.Teaser, error)
}

// The publishing module's posts.
type Posts interface {
	// The battles the posts since then tease.
	Teased(ctx context.Context, since time.Time) ([]teasers.BattleID, error)
}

// Where the renderer reads each teaser.
type Store interface {
	// The folder's name.
	Keep(ctx context.Context, teaser teasers.Teaser) (string, error)
}

// A teaser to render, in its folder.
type Due struct {
	Folder string
	Teaser teasers.Teaser
}

type Executor interface {
	Execute(ctx context.Context) ([]Due, error)
}

func New(battles Battles, posts Posts, store Store, clock cptime.Clock) *UseCase {
	return &UseCase{battles: battles, posts: posts, store: store, clock: clock}
}

type UseCase struct {
	battles Battles
	posts   Posts
	store   Store
	clock   cptime.Clock
}

var _ Executor = (*UseCase)(nil)

// Each battle within the next run's reach that no post teases yet, kept for the renderer.
func (u *UseCase) Execute(ctx context.Context) ([]Due, error) {
	listed, err := u.battles.Teasers(ctx, teasers.Ahead)
	if err != nil {
		return nil, fmt.Errorf("failed to list the battles to come: %w", err)
	}
	teased, err := u.posts.Teased(ctx, teasers.TeasedSince(u.clock.Now()))
	if err != nil {
		return nil, fmt.Errorf("failed to read the battles teased already: %w", err)
	}

	due := teasers.Due(listed, teased)
	kept := make([]Due, 0, len(due))
	for _, teaser := range due {
		folder, err := u.store.Keep(ctx, teaser)
		if err != nil {
			return nil, err //nolint:wrapcheck // the store already named the teaser it could not keep.
		}
		kept = append(kept, Due{Folder: folder, Teaser: teaser})
	}
	return kept, nil
}
