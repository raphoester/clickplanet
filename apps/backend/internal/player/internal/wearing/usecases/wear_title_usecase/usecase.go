package wear_title_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Wardrobe interface {
	Wear(ctx context.Context, account players.AccountID, title titles.ID, at time.Time) error
	Showcase(ctx context.Context, account players.AccountID) (wearing.Showcase, error)
}

type UseCase struct {
	wardrobe Wardrobe
	clock    cptime.Clock
}

func New(wardrobe Wardrobe, clock cptime.Clock) *UseCase {
	return &UseCase{wardrobe: wardrobe, clock: clock}
}

func (u *UseCase) Execute(ctx context.Context, account players.AccountID, title titles.ID) (titles.Standing, error) {
	if err := u.wardrobe.Wear(ctx, account, title, u.clock.Now()); err != nil {
		return titles.Standing{}, fmt.Errorf("failed to wear the title: %w", err)
	}

	showcase, err := u.wardrobe.Showcase(ctx, account)
	if err != nil {
		return titles.Standing{}, fmt.Errorf("failed to read the worn title: %w", err)
	}
	return showcase.Worn(), nil
}
