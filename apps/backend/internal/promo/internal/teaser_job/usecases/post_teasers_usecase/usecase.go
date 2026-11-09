package post_teasers_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teaser_job"
)

// What the renderer made.
type Renders interface {
	Rendered(ctx context.Context) ([]teaser_job.Rendered, error)
}

// The posting service's storage, through the publishing module's upload links.
type Storage interface {
	// *teaser_job.BusyError: try later.
	Upload(ctx context.Context, file teaser_job.File, contentType string) (teaser_job.MediaURL, error)
}

// Promo's PostTeaser.
type Promo interface {
	// *teaser_job.BusyError: try later.
	Post(ctx context.Context, battle teaser_job.BattleID, media teaser_job.Media) (teaser_job.Outcome, error)
}

type Executor interface {
	// A teaser that was not posted is in the report, not in the error.
	Execute(ctx context.Context) (teaser_job.Report, error)
}

func New(renders Renders, storage Storage, promo Promo, backoff teaser_job.Backoff, pause teaser_job.Pause) *UseCase {
	return &UseCase{renders: renders, storage: storage, promo: promo, backoff: backoff, pause: pause}
}

type UseCase struct {
	renders Renders
	storage Storage
	promo   Promo
	backoff teaser_job.Backoff
	pause   teaser_job.Pause
}

var _ Executor = (*UseCase)(nil)

// Each render uploaded and handed to promo, one after the other, so the calendar holds each post before the next.
func (u *UseCase) Execute(ctx context.Context) (teaser_job.Report, error) {
	rendered, err := u.renders.Rendered(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read the rendered teasers: %w", err)
	}
	report := make(teaser_job.Report, 0, len(rendered))
	for _, one := range rendered {
		outcome, err := u.post(ctx, one)
		if err != nil {
			outcome = teaser_job.Outcome{Battle: one.Battle, Result: teaser_job.Failed, Why: err.Error()}
		}
		report = append(report, outcome)
	}
	return report, nil
}

func (u *UseCase) post(ctx context.Context, one teaser_job.Rendered) (teaser_job.Outcome, error) {
	var media teaser_job.Media
	for _, photo := range one.Photos {
		address, err := retry(ctx, u, func() (teaser_job.MediaURL, error) { return u.storage.Upload(ctx, photo, "image/jpeg") })
		if err != nil {
			return teaser_job.Outcome{}, err
		}
		media.Photos = append(media.Photos, address)
	}
	video, err := retry(ctx, u, func() (teaser_job.MediaURL, error) { return u.storage.Upload(ctx, one.Video, "video/mp4") })
	if err != nil {
		return teaser_job.Outcome{}, err
	}
	media.Video = video
	return retry(ctx, u, func() (teaser_job.Outcome, error) { return u.promo.Post(ctx, one.Battle, media) })
}

// The call again after each wait the backoff takes, while it is busy.
func retry[T any](ctx context.Context, u *UseCase, call func() (T, error)) (T, error) {
	for failed := 1; ; failed++ {
		answer, err := call()
		var busy *teaser_job.BusyError
		if !errors.As(err, &busy) {
			return answer, err
		}
		wait, again := u.backoff.Wait(failed, busy.RetryAfter)
		if !again {
			return answer, err
		}
		if err := u.pause.Wait(ctx, wait); err != nil {
			return answer, err //nolint:wrapcheck // a pure delegation: the pause only ends early with the context.
		}
	}
}
