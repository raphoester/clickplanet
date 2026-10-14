package choose_clips_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Shortlists interface {
	Shortlist(ctx context.Context) (posts.Shortlist, error)
}

type History interface {
	// Every post with a story key that goes out at since or later, and maybe some before.
	Told(ctx context.Context, since time.Time) ([]posts.Telling, error)
}

type Executor interface {
	Execute(ctx context.Context) (posts.Choice, error)
}

func New(shortlists Shortlists, history History, clock cptime.Clock, perRun int) *UseCase {
	return &UseCase{shortlists: shortlists, history: history, clock: clock, perRun: perRun}
}

type UseCase struct {
	shortlists Shortlists
	history    History
	clock      cptime.Clock
	perRun     int
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context) (posts.Choice, error) {
	shortlist, err := u.shortlists.Shortlist(ctx)
	if err != nil {
		return posts.Choice{}, fmt.Errorf("failed to read the stories worth a clip: %w", err)
	}

	now := u.clock.Now()
	told, err := u.history.Told(ctx, posts.ToldSince(now))
	if err != nil {
		return posts.Choice{}, fmt.Errorf("failed to read the stories already told: %w", err)
	}
	return shortlist.Best(u.perRun, told, now), nil
}
