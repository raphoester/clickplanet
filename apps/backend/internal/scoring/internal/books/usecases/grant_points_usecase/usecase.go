package grant_points_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
)

type Grants interface {
	Keep(ctx context.Context, grant books.Grant) (bool, error)
}

type Command struct {
	Grant   books.GrantID
	Country books.Country
	Points  uint32
	At      time.Time
}

func New(store Grants) *UseCase {
	return &UseCase{store: store}
}

type UseCase struct {
	store Grants
}

// Execute answers whether it kept the grant: one kept before is left as it is.
func (u *UseCase) Execute(ctx context.Context, command Command) (bool, error) {
	grant, err := books.NewGrant(command.Grant, command.Country, command.Points, command.At)
	if err != nil {
		return false, err //nolint:wrapcheck // the domain named it.
	}
	kept, err := u.store.Keep(ctx, grant)
	if err != nil {
		return false, fmt.Errorf("failed to keep grant %s: %w", command.Grant, err)
	}
	return kept, nil
}
