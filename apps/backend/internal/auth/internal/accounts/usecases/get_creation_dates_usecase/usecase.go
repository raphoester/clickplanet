package get_creation_dates_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

type CreationDates interface {
	CreationDates(ctx context.Context, accounts []accounts.AccountID) (map[accounts.AccountID]time.Time, error)
}

type UseCase struct {
	dates CreationDates
}

func New(dates CreationDates) *UseCase {
	return &UseCase{dates: dates}
}

func (u *UseCase) Execute(ctx context.Context, asked []accounts.AccountID) (map[accounts.AccountID]time.Time, error) {
	dates, err := u.dates.CreationDates(ctx, asked)
	if err != nil {
		return nil, fmt.Errorf("failed to read the creation dates: %w", err)
	}
	return dates, nil
}
