package standings

import (
	"context"
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
)

var (
	ErrNotStarted = errors.New("the standings do not count the takes yet")
	ErrStarted    = errors.New("the standings already count the takes")
	ErrMoved      = errors.New("the standings moved past where the batch was read from")
)

type Store interface {
	Position(ctx context.Context) (Position, error)
	Begin(ctx context.Context, start Position) error
	Count(ctx context.Context, batch Batch, seasons calendar.Calendar) error
	Rewind(ctx context.Context) (Position, error)
	DeleteAccount(ctx context.Context, account AccountID) error
}
