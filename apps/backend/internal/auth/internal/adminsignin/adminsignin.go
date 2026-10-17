package adminsignin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
)

const (
	CookieName     = "cp_admin"
	FlowCookieName = "cp_admin_oauth"

	DefaultTTL = 12 * time.Hour
)

var (
	ErrSignInOff       = errors.New("the admin sign-in is off on this server")
	ErrNotAnOperator   = errors.New("this Google account does not operate")
	ErrUnknownIdentity = errors.New("no account holds this identity")
)

type Candidate struct {
	account  accounts.AccountID
	operates bool
}

func CandidateOf(account accounts.AccountID, operates bool) Candidate {
	return Candidate{account: account, operates: operates}
}

func (c Candidate) SessionError() error {
	if !c.operates {
		return ErrNotAnOperator
	}
	return nil
}

type Snapshot struct {
	TokenHash accounts.TokenHash
	Account   accounts.AccountID
	CreatedAt time.Time
	ExpiresAt time.Time
}

// Never extended: an admin session ends when its lifetime does, and the operator signs in again.
type Session struct {
	token     *accounts.Token
	account   accounts.AccountID
	createdAt time.Time
	expiresAt time.Time
}

func (s *Session) TakeSnapshot() Snapshot {
	return Snapshot{TokenHash: s.token.Hash(), Account: s.account, CreatedAt: s.createdAt, ExpiresAt: s.expiresAt}
}

func (s *Session) Cookie(now time.Time) string {
	return s.token.StrictCookie(CookieName, s.expiresAt, now)
}

func ExpiredCookie() string {
	return accounts.ExpiredStrictCookie(CookieName)
}

func FlowCookie(flow *signin.Flow, sealed string, now time.Time) string {
	return accounts.Cookie(FlowCookieName, sealed, flow.ExpiresAt(), now)
}

func ExpiredFlowCookie() string {
	return accounts.ExpiredCookie(FlowCookieName)
}

func NewIssuer(tokens accounts.TokenGenerator, ttl time.Duration) *Issuer {
	return &Issuer{tokens: tokens, ttl: ttl}
}

type Issuer struct {
	tokens accounts.TokenGenerator
	ttl    time.Duration
}

func (i *Issuer) Session(candidate Candidate, now time.Time) (*Session, error) {
	if err := candidate.SessionError(); err != nil {
		return nil, err
	}

	token, err := i.tokens.NewToken()
	if err != nil {
		return nil, fmt.Errorf("failed to draw a session token: %w", err)
	}

	return &Session{token: token, account: candidate.account, createdAt: now, expiresAt: now.Add(i.ttl)}, nil
}

type Store interface {
	Candidate(ctx context.Context, provider string, subject string) (Candidate, error)
	SaveSession(ctx context.Context, session Snapshot) error
	DeleteSession(ctx context.Context, tokenHash accounts.TokenHash) error
}
