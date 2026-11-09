package teasers

import (
	"context"
	"errors"
	"time"
)

// How many slots in a row may be taken before the post is busy: another post made at the same time, not a loop.
const maxTaken = 3

// The server's publishing module.
type Publishing interface {
	// The first slot from notBefore to by at which every one of the platforms takes the post: false when none does.
	Slot(ctx context.Context, platforms []Platform, notBefore, by time.Time) (Slot, bool, error)
	// The post at its slot. *BusyError: try later. ErrTaken: plan again. *RefusedError: never.
	Schedule(ctx context.Context, post Post) (Receipt, error)
}

func NewPoster(publishing Publishing) Poster {
	return Poster{publishing: publishing}
}

// Posts a teaser at the first slot of its window, and never waits: a busy service is the caller's to wait for.
type Poster struct {
	publishing Publishing
}

// Held when no platform takes it before its battle; a slot taken since it was planned is planned again. An error is
// for a call that failed otherwise, which the publishing module named.
func (p Poster) Post(ctx context.Context, post Post) (Outcome, error) {
	notBefore, by := post.teaser.Window()
	for taken := 0; ; taken++ {
		slot, found, err := p.slot(ctx, notBefore, by)
		if err != nil {
			return outcomeOf(err)
		}
		if !found {
			return NotPosted(Held, "no platform takes it before its battle: their days are full, or their hours past"), nil
		}
		post = post.At(slot)
		receipt, err := p.publishing.Schedule(ctx, post)
		if errors.Is(err, ErrTaken) && taken < maxTaken {
			continue
		}
		if err != nil {
			return outcomeOf(err)
		}
		return PostedOf(post, receipt), nil
	}
}

// The slot of the first of the choices that takes the post by its battle.
func (p Poster) slot(ctx context.Context, notBefore, by time.Time) (Slot, bool, error) {
	for _, platforms := range choices {
		slot, found, err := p.publishing.Slot(ctx, platforms, notBefore, by)
		if err != nil || found {
			return slot, found, err //nolint:wrapcheck // a pure delegation: the adapter already named what failed.
		}
	}
	return Slot{}, false, nil
}

func outcomeOf(err error) (Outcome, error) {
	var busy *BusyError
	var refused *RefusedError
	switch {
	case errors.As(err, &busy):
		return BusyOf(busy), nil
	case errors.Is(err, ErrTaken):
		return BusyOf(&BusyError{Reason: "the slot was taken again and again: another post is being made"}), nil
	case errors.As(err, &refused):
		return NotPosted(Failed, refused.Reason), nil
	default:
		return Outcome{}, err
	}
}
