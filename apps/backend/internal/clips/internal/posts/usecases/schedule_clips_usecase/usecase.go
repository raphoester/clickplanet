package schedule_clips_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

var ErrNoClip = errors.New("no clip was rendered")

type Clips interface {
	Rendered(ctx context.Context) ([]posts.Clip, error)
}

type Sender interface {
	// At the slot the calendar gives it: held when no platform takes it. It answers the post as it was sent, at its slot.
	Send(ctx context.Context, post posts.Post) (posts.Post, posts.Scheduled, bool, error)
}

type Executor interface {
	// A clip the service refused is in the report, not in the error.
	Execute(ctx context.Context) (posts.Report, error)
}

func New(clips Clips, plan posts.Plan, dice posts.Dice, sender Sender) *UseCase {
	return &UseCase{clips: clips, plan: plan, dice: dice, sender: sender}
}

type UseCase struct {
	clips  Clips
	plan   posts.Plan
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

	// Each post is planned once the one before it is sent, so the calendar holds it: a wait that moved it moves the next
	// one too.
	report := posts.Report{}
	ranked := posts.Ranked(rendered)
	for i, clip := range ranked {
		post := u.plan.Post(clip, i+1, len(ranked), u.dice)
		sent, scheduled, held, err := u.sender.Send(ctx, post)
		if held {
			report = report.WithHeld(clip)
			continue
		}
		report = report.With(sent, scheduled, err)
	}
	return report, nil
}
