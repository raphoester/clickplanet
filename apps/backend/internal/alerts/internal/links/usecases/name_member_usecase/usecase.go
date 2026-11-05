package name_member_usecase

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
)

type Namer interface {
	Name(ctx context.Context, account links.AccountID) (bool, error)
}

// A 429 waited out twice fits.
const timeout = 15 * time.Second

func New(namer Namer) *UseCase {
	return &UseCase{namer: namer}
}

type UseCase struct {
	namer Namer
}

// The account's Discord user, when linked, wears the name player shows for the account now, or none: see links.Namer.
func (u *UseCase) Execute(ctx context.Context, account links.AccountID) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_, err := u.namer.Name(ctx, account)
	return err //nolint:wrapcheck // the namer named it.
}
