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
	Send(ctx context.Context, post posts.Post) (posts.PostID, error)
}

type Summary interface {
	Write(report posts.Report) error
}

type Executor interface {
	// The report says what became of each clip, also when a clip was not scheduled and the error says so.
	Execute(ctx context.Context) (posts.Report, error)
}

func New(clips Clips, cadence posts.Cadence, clock cptime.Clock, sender Sender, summary Summary) *UseCase {
	return &UseCase{clips: clips, cadence: cadence, clock: clock, sender: sender, summary: summary}
}

type UseCase struct {
	clips   Clips
	cadence posts.Cadence
	clock   cptime.Clock
	sender  Sender
	summary Summary
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
	for _, post := range u.cadence.Posts(rendered, u.clock.Now()) {
		id, err := u.sender.Send(ctx, post)
		report = report.With(post, id, err)
	}

	if err := u.summary.Write(report); err != nil {
		return report, fmt.Errorf("failed to write the summary of the run: %w", err)
	}
	return report, report.UnsentError() //nolint:wrapcheck // a pure delegation: the report already named what failed.
}
