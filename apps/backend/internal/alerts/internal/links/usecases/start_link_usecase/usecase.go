package start_link_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Store interface {
	SaveFlow(ctx context.Context, flow links.Flow) error
}

type Authorizer interface {
	AuthorizationURL(state string, challenge string) string
}

const timeout = 5 * time.Second

func New(secrets links.Secrets, clock cptime.Clock, store Store, authorizer Authorizer) *UseCase {
	return &UseCase{secrets: secrets, clock: clock, store: store, authorizer: authorizer}
}

type UseCase struct {
	secrets    links.Secrets
	clock      cptime.Clock
	store      Store
	authorizer Authorizer
}

// The Discord page to send the player to.
func (u *UseCase) Execute(ctx context.Context, account links.AccountID) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	flow, err := links.NewFlow(account, u.secrets, u.clock.Now())
	if err != nil {
		return "", fmt.Errorf("failed to start a link: %w", err)
	}
	if err := u.store.SaveFlow(ctx, flow); err != nil {
		return "", fmt.Errorf("failed to keep the link flow: %w", err)
	}
	authorization := flow.Authorization()
	return u.authorizer.AuthorizationURL(authorization.State, authorization.Challenge), nil
}
