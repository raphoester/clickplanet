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

type Feeds interface {
	// Every post, by hand or not, that goes out at since or later on each platform, and maybe some before.
	Posted(ctx context.Context, since time.Time) (posts.Feed, error)
}

// How many clips a run makes: perRun at most, within each platform's day.
type Room struct {
	PerRun    int
	Budget    posts.Budget
	Platforms []posts.Platform
}

type Executor interface {
	Execute(ctx context.Context) (posts.Choice, error)
}

// dice: the coins that put each story chosen in an arm of every experiment.
func New(
	shortlists Shortlists, history History, feeds Feeds, clock cptime.Clock, room Room, experiments posts.Experiments,
	dice posts.Dice,
) *UseCase {
	return &UseCase{
		shortlists: shortlists, history: history, feeds: feeds, clock: clock, room: room, experiments: experiments, dice: dice,
	}
}

type UseCase struct {
	shortlists  Shortlists
	history     History
	feeds       Feeds
	clock       cptime.Clock
	room        Room
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
	feed, err := u.feeds.Posted(ctx, posts.PostedSince(now))
	if err != nil {
		return posts.Choice{}, fmt.Errorf("failed to read what each platform holds: %w", err)
	}
	choice := shortlist.Best(u.room.Budget.Clips(u.room.PerRun, u.room.Platforms, feed, now), told, now)
	choice.Assigned = u.experiments.AssignEach(choice.Chosen, u.dice)
	return choice, nil
}
