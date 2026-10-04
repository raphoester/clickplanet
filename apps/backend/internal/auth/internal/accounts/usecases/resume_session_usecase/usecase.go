package resume_session_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Resumer interface {
	Resume(ctx context.Context, cookieHeader string, now time.Time) (accounts.Entry, error)
}

type Minter interface {
	Mint(ip string, holder cpsession.Holder, now time.Time) (*cpsession.Token, error)
}

type In struct {
	IP           string
	CookieHeader string
}

// Token is nil when the cookie holds no live session: only a Turnstile check starts one.
type Out struct {
	Token     *cpsession.Token
	SetCookie string
}

var ErrNoAddress = errors.New("the request carries no source address")

func New(resumer Resumer, minter Minter, clock cptime.Clock) *UseCase {
	return &UseCase{resumer: resumer, minter: minter, clock: clock}
}

type UseCase struct {
	resumer Resumer
	minter  Minter
	clock   cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, in In) (*Out, error) {
	// An empty IP binds the token to "", which every other caller would verify against too.
	if in.IP == "" {
		return nil, ErrNoAddress
	}

	now := u.clock.Now()
	entry, err := u.resumer.Resume(ctx, in.CookieHeader, now)
	if accounts.Ended(err) {
		return &Out{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find the caller's account: %w", err)
	}

	token, err := u.minter.Mint(in.IP, cpsession.Holder{Account: entry.Account(), Linked: entry.Linked()}, now)
	if err != nil {
		return nil, fmt.Errorf("failed to mint the identity token: %w", err)
	}
	return &Out{Token: token, SetCookie: entry.SetCookie()}, nil
}
