package register_client_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Sealer interface {
	SealedClient(client *connections.Client) (string, error)
}

func New(redirects connections.Redirects, sealer Sealer, clock cptime.Clock) *UseCase {
	return &UseCase{redirects: redirects, sealer: sealer, clock: clock}
}

type UseCase struct {
	redirects connections.Redirects
	sealer    Sealer
	clock     cptime.Clock
}

type In struct {
	Name         string
	RedirectURIs []string
}

type Out struct {
	ClientID string
	Client   connections.ClientSnapshot
}

// RFC 7591, with nothing stored: the client id is the registration, sealed.
func (u *UseCase) Execute(_ context.Context, in In) (Out, error) {
	client, err := connections.NewClient(in.Name, in.RedirectURIs, u.redirects, u.clock.Now())
	if err != nil {
		return Out{}, fmt.Errorf("failed to register the client: %w", err)
	}

	id, err := u.sealer.SealedClient(client)
	if err != nil {
		return Out{}, fmt.Errorf("failed to seal the client: %w", err)
	}

	return Out{ClientID: id, Client: client.TakeSnapshot()}, nil
}
