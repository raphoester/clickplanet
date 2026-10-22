package authorize_usecase

import (
	"context"
	"fmt"
	"net/url"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Sealer interface {
	OpenedClient(sealed string) (*connections.Client, error)
	SealedRequest(request *connections.Request) (string, error)
}

// The admin page's consent page, which the operator's browser is sent to with the request.
func New(consentURL string, sealer Sealer, clock cptime.Clock) *UseCase {
	return &UseCase{consentURL: consentURL, sealer: sealer, clock: clock}
}

type UseCase struct {
	consentURL string
	sealer     Sealer
	clock      cptime.Clock
}

type In struct {
	ClientID string
	Ask      connections.Ask
}

// Where the browser goes next: the consent page. A *connections.Refusal carries where to send it back to the app
// instead; ErrInvalidClient and ErrInvalidRedirect leave it here.
func (u *UseCase) Execute(_ context.Context, in In) (string, error) {
	client, err := u.sealer.OpenedClient(in.ClientID)
	if err != nil {
		return "", fmt.Errorf("failed to read the client: %w", err)
	}

	request, err := client.Request(in.ClientID, in.Ask, u.clock.Now())
	if err != nil {
		return "", fmt.Errorf("failed to read the request: %w", err)
	}

	sealed, err := u.sealer.SealedRequest(request)
	if err != nil {
		return "", fmt.Errorf("failed to seal the request: %w", err)
	}

	return u.consentURL + "?" + url.Values{"request": {sealed}}.Encode(), nil
}
