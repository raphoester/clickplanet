package schedule_clips_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var ErrNoClip = errors.New("no clip was rendered")

type Clips interface {
	Rendered(ctx context.Context) ([]posts.Clip, error)
}

type Sender interface {
	// It answers the post as it was sent, which may be later than planned.
	Send(ctx context.Context, post posts.Post) (posts.Post, posts.Scheduled, error)
}

type Feeds interface {
	// Every post, by hand or not, that goes out at since or later on each platform, and maybe some before.
	Posted(ctx context.Context, since time.Time) (posts.Feed, error)
}

type Executor interface {
	// A clip the service refused is in the report, not in the error.
	Execute(ctx context.Context) (posts.Report, error)
}

func New(clips Clips, feeds Feeds, plan posts.Plan, clock cptime.Clock, dice posts.Dice, sender Sender) *UseCase {
	return &UseCase{clips: clips, feeds: feeds, plan: plan, clock: clock, dice: dice, sender: sender}
}

type UseCase struct {
	clips  Clips
	feeds  Feeds
	plan   posts.Plan
	clock  cptime.Clock
	dice   posts.Dice
	sender Sender
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context) (posts.Report, error) {
	rendered, err := u.clips.Rendered(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read the rendered clips: %w", err)
	}
	if len(rendered) == 0 {
		return nil, ErrNoClip
	}

	feed, err := u.feeds.Posted(ctx, posts.PostedSince(u.clock.Now()))
	if err != nil {
		return nil, fmt.Errorf("failed to read what each platform holds: %w", err)
	}

	// Each post is planned once the one before it is sent: a wait that moved it moves the next one too.
	report := posts.Report{}
	ranked := posts.Ranked(rendered)
	for i, clip := range ranked {
		post, room := u.plan.Next(clip, i+1, len(ranked), feed, u.clock.Now(), u.dice)
		if !room {
			report = report.WithHeld(clip)
			continue
		}
		sent, scheduled, err := u.sender.Send(ctx, post)
		report = report.With(sent, scheduled, err)
		feed = feed.With(sent, scheduled)
	}
	return report, nil
}
