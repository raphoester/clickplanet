package post_teasers_usecase

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

// What the renderer made.
type Renders interface {
	Rendered(ctx context.Context) ([]teasers.Rendered, error)
}

type Sender interface {
	// At the first slot of its window the calendar gives it: held when no platform takes it before its battle. It
	// answers the post as it was sent, at its slot.
	Send(ctx context.Context, post teasers.Post) (teasers.Post, teasers.Receipt, bool, error)
}

type Executor interface {
	// A teaser the service refused is in the report, not in the error.
	Execute(ctx context.Context) (teasers.Report, error)
}

func New(battles Battles, posts Posts, renders Renders, sender Sender, clock cptime.Clock) *UseCase {
	return &UseCase{battles: battles, posts: posts, renders: renders, sender: sender, clock: clock}
}

type UseCase struct {
	battles Battles
	posts   Posts
	renders Renders
	sender  Sender
	clock   cptime.Clock
}

var _ Executor = (*UseCase)(nil)

// Each battle still due, read again now: the battles and the posts may have changed while the cards were rendered.
// One rendered and no longer due, or due and not rendered, is skipped, and the next run sees it again.
func (u *UseCase) Execute(ctx context.Context) (teasers.Report, error) {
	rendered, err := u.renders.Rendered(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read the rendered teasers: %w", err)
	}
	listed, err := u.battles.Teasers(ctx, teasers.Ahead)
	if err != nil {
		return nil, fmt.Errorf("failed to list the battles to come: %w", err)
	}
	teased, err := u.posts.Teased(ctx, teasers.TeasedSince(u.clock.Now()))
	if err != nil {
		return nil, fmt.Errorf("failed to read the battles teased already: %w", err)
	}

	media := make(map[teasers.BattleID]teasers.Media, len(rendered))
	for _, one := range rendered {
		media[one.Battle] = one.Media
	}
	report := teasers.Report{}
	for _, teaser := range teasers.Due(listed, teased) {
		files, ok := media[teaser.Battle]
		if !ok {
			report = report.WithSkipped(teaser.Battle, teaser.Words().Title, "not rendered: it was not due when the run began, or its render failed")
			continue
		}
		delete(media, teaser.Battle)
		// One after the other: the calendar holds each post before the next is planned.
		sent, receipt, held, err := u.sender.Send(ctx, teasers.Post{Teaser: teaser, Media: files})
		report = report.With(sent, receipt, held, err)
	}
	for _, one := range rendered {
		if _, left := media[one.Battle]; left {
			report = report.WithSkipped(one.Battle, string(one.Battle), "no longer due: teased already, or no longer on the schedule")
		}
	}
	return report, nil
}
