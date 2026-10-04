package takes

import (
	"context"
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

var (
	ErrNotStarted = errors.New("the stats do not count the takes yet")
	ErrStarted    = errors.New("the stats already count the takes")
	ErrMoved      = errors.New("the stats moved past where the batch was read from")
)

type Store interface {
	Position(ctx context.Context) (Position, error)
	Begin(ctx context.Context, start Position) error
	Count(ctx context.Context, batch Batch) error
	Rewind(ctx context.Context) (Position, error)
	DeleteAccount(ctx context.Context, account players.AccountID) error
}
