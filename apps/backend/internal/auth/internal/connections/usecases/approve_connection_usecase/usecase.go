package approve_connection_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Sealer interface {
	OpenedRequest(sealed string) (*connections.Request, error)
	SealedCode(code *connections.Code) (string, error)
}

type IDs interface {
	NewSecret() (string, error)
}

func New(issuer string, sealer Sealer, ids IDs, clock cptime.Clock) *UseCase {
	return &UseCase{issuer: issuer, sealer: sealer, ids: ids, clock: clock}
}

type UseCase struct {
	issuer string
	sealer Sealer
	ids    IDs
	clock  cptime.Clock
}

type In struct {
	Request  string
	Operator accounts.AccountID
}

type Out struct {
	RedirectURL string
	Client      string
	Host        string
}

// The operator lets the app in: a code for the operator's account, which the app's browser carries back to it.
func (u *UseCase) Execute(_ context.Context, in In) (Out, error) {
	now := u.clock.Now()

	request, err := u.sealer.OpenedRequest(in.Request)
	if err != nil {
		return Out{}, fmt.Errorf("failed to read the request: %w", err)
	}
	if err := request.ExpiryError(now); err != nil {
		return Out{}, fmt.Errorf("failed to read the request: %w", err)
	}

	id, err := u.ids.NewSecret()
	if err != nil {
		return Out{}, fmt.Errorf("failed to draw the code's id: %w", err)
	}
	code, err := u.sealer.SealedCode(request.Approved(in.Operator, id, now))
	if err != nil {
		return Out{}, fmt.Errorf("failed to seal the code: %w", err)
	}

	return Out{RedirectURL: request.ApprovedURL(code, u.issuer), Client: request.Client(), Host: request.RedirectHost()}, nil
}
