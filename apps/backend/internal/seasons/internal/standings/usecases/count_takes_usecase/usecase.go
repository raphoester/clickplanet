package count_takes_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
)

type Feed interface {
	Start(ctx context.Context) (standings.Position, error)
	Batch(ctx context.Context, from standings.Position) (standings.Batch, error)
}

type Tally interface {
	Position(ctx context.Context) (standings.Position, error)
	Begin(ctx context.Context, start standings.Position) error
	Count(ctx context.Context, batch standings.Batch, seasons calendar.Calendar) error
}

type Executor interface {
	Execute(ctx context.Context) (Out, error)
}

type Out struct {
	Began    bool
	From     standings.Position
	Takes    int
	CaughtUp bool
}

type UseCase struct {
	feed    Feed
	tally   Tally
	seasons calendar.Calendar
}

var _ Executor = (*UseCase)(nil)

func New(feed Feed, tally Tally, seasons calendar.Calendar) *UseCase {
	return &UseCase{feed: feed, tally: tally, seasons: seasons}
}

func (u *UseCase) Execute(ctx context.Context) (Out, error) {
	from, began, err := u.position(ctx)
	if err != nil {
		return Out{}, err
	}
	out := Out{Began: began, From: from}

	batch, err := u.feed.Batch(ctx, from)
	if err != nil {
		return out, fmt.Errorf("failed to read the takes from %d: %w", from, err)
	}
	if batch.Empty() {
		out.CaughtUp = true
		return out, nil
	}

	err = u.tally.Count(ctx, batch, u.seasons)
	if errors.Is(err, standings.ErrMoved) {
		return out, nil
	}
	if err != nil {
		return out, fmt.Errorf("failed to count the takes from %d: %w", from, err)
	}

	out.Takes = batch.Len()
	return out, nil
}

func (u *UseCase) position(ctx context.Context) (standings.Position, bool, error) {
	position, err := u.tally.Position(ctx)
	if err == nil {
		return position, false, nil
	}
	if !errors.Is(err, standings.ErrNotStarted) {
		return 0, false, fmt.Errorf("failed to read where the standings are counted to: %w", err)
	}

	start, err := u.feed.Start(ctx)
	if err != nil {
		return 0, false, fmt.Errorf("failed to ask where the feed starts: %w", err)
	}
	if err := u.tally.Begin(ctx, start); err != nil {
		return 0, false, fmt.Errorf("failed to begin counting at %d: %w", start, err)
	}
	return start, true, nil
}
