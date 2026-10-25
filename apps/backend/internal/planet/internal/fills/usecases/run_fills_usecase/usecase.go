package run_fills_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/fills"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const PageSize = 500

type Fills interface {
	Unfinished(ctx context.Context) (fills.Fill, error)
	Save(ctx context.Context, fill fills.Fill) error
}

type Players interface {
	Players(ctx context.Context, page fills.Page) ([]bonuses.Holder, error)
}

type Hands interface {
	Fill(ctx context.Context, holder bonuses.Holder, fill bonuses.FillID) error
}

type Notice interface {
	PublishFilled()
}

type Executor interface {
	// Execute runs every fill not finished to its end, oldest first, and answers the ones it finished.
	Execute(ctx context.Context) ([]fills.Fill, error)
}

func New(kept Fills, players Players, hands Hands, notice Notice, clock cptime.Clock, pageSize int) *UseCase {
	return &UseCase{fills: kept, players: players, hands: hands, notice: notice, clock: clock, pageSize: pageSize}
}

// UseCase runs one fill at a time, which is what lets a hand remember only the last fill it had.
type UseCase struct {
	fills   Fills
	players Players
	hands   Hands
	notice  Notice
	clock   cptime.Clock

	pageSize int
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context) ([]fills.Fill, error) {
	var finished []fills.Fill
	for {
		fill, err := u.fills.Unfinished(ctx)
		if err != nil {
			return finished, fmt.Errorf("failed to read the fill to run: %w", err)
		}
		if fill == (fills.Fill{}) {
			return finished, nil
		}

		for fill.Running() {
			if fill, err = u.next(ctx, fill); err != nil {
				return finished, err
			}
		}
		finished = append(finished, fill)
	}
}

func (u *UseCase) next(ctx context.Context, fill fills.Fill) (fills.Fill, error) {
	page := fill.NextPage(u.pageSize)
	holders, err := u.players.Players(ctx, page)
	if err != nil {
		return fill, fmt.Errorf("failed to read whom the fill %q gives to: %w", fill.ID(), err)
	}

	for _, holder := range holders {
		if err := u.hands.Fill(ctx, holder, fill.ID()); err != nil {
			return fill, err //nolint:wrapcheck // the charges say what failed.
		}
	}

	fill = fill.Past(page, holders, u.clock.Now())
	if err := u.fills.Save(ctx, fill); err != nil {
		return fill, fmt.Errorf("failed to keep how far the fill %q got: %w", fill.ID(), err)
	}
	if len(holders) > 0 {
		u.notice.PublishFilled()
	}

	return fill, nil
}
