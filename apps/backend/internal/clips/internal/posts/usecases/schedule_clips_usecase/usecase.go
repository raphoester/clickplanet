package schedule_clips_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var ErrNoClip = errors.New("no clip was rendered")

type Clips interface {
	Rendered(ctx context.Context) ([]posts.Clip, error)
}

type Sender interface {
	Send(ctx context.Context, post posts.Post) (posts.Scheduled, error)
}

type Executor interface {
	// A clip the service refused is in the report, not in the error.
	Execute(ctx context.Context) (posts.Report, error)
}

func New(clips Clips, plan posts.Plan, clock cptime.Clock, dice posts.Dice, sender Sender) *UseCase {
	return &UseCase{clips: clips, plan: plan, clock: clock, dice: dice, sender: sender}
}

type UseCase struct {
	clips  Clips
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

	report := posts.Report{}
	for _, post := range u.plan.Posts(rendered, u.clock.Now(), u.dice) {
		scheduled, err := u.sender.Send(ctx, post)
		report = report.With(post, scheduled, err)
	}
	return report, nil
}
