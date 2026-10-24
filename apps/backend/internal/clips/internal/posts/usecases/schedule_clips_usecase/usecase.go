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

// experiments: the ones that run, which every clip's arms are checked against before any is posted.
func New(
	clips Clips, plan posts.Plan, clock cptime.Clock, dice posts.Dice, sender Sender, experiments posts.Experiments,
) *UseCase {
	return &UseCase{clips: clips, plan: plan, clock: clock, dice: dice, sender: sender, experiments: experiments}
}

type UseCase struct {
	clips       Clips
	plan        posts.Plan
	clock       cptime.Clock
	dice        posts.Dice
	sender      Sender
	experiments posts.Experiments
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
	// Before any post: a clip in an arm nobody runs fails the run, rather than be counted wrong once it is out.
	for _, clip := range rendered {
		snapshot := clip.TakeSnapshot()
		if err := u.experiments.Check(snapshot.Variants); err != nil {
			return nil, fmt.Errorf("the clip %q: %w", snapshot.Title, err)
		}
	}

	report := posts.Report{}
	for _, post := range u.plan.Posts(rendered, u.clock.Now(), u.dice) {
		scheduled, err := u.sender.Send(ctx, post)
		report = report.With(post, scheduled, err)
	}
	return report, nil
}
