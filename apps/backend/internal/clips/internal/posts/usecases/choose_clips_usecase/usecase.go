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

// dice: the coins that put each story chosen in an arm of every experiment.
func New(
	shortlists Shortlists, history History, clock cptime.Clock, perRun int, experiments posts.Experiments, dice posts.Dice,
) *UseCase {
	return &UseCase{shortlists: shortlists, history: history, clock: clock, perRun: perRun, experiments: experiments, dice: dice}
}

type UseCase struct {
	shortlists  Shortlists
	history     History
	clock       cptime.Clock
	perRun      int
	experiments posts.Experiments
	dice        posts.Dice
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
	choice := shortlist.Best(u.perRun, told, now)
	choice.Assigned = u.experiments.AssignEach(choice.Chosen, u.dice)
	return choice, nil
}
