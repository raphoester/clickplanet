package describe_connection_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Sealer interface {
	OpenedRequest(sealed string) (*connections.Request, error)
}

func New(sealer Sealer, clock cptime.Clock) *UseCase {
	return &UseCase{sealer: sealer, clock: clock}
}

type UseCase struct {
	sealer Sealer
	clock  cptime.Clock
}

type Out struct {
	Client       string
	RedirectHost string
	DeniedURL    string
}

// What the operator is asked to let in. It decides nothing: the approval opens the request again.
func (u *UseCase) Execute(_ context.Context, sealed string) (Out, error) {
	request, err := u.sealer.OpenedRequest(sealed)
	if err != nil {
		return Out{}, fmt.Errorf("failed to read the request: %w", err)
	}
	if err := request.ExpiryError(u.clock.Now()); err != nil {
		return Out{}, fmt.Errorf("failed to read the request: %w", err)
	}

	return Out{Client: request.Client(), RedirectHost: request.RedirectHost(), DeniedURL: request.DeniedURL()}, nil
}
