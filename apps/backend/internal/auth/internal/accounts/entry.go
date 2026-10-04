package accounts

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// The account a browser comes in as, and the cookie to set: none when the one it sent still stands.
type Entry struct {
	account   AccountID
	linked    bool
	setCookie string
}

func (e Entry) Account() AccountID { return e.account }

func (e Entry) Linked() bool { return e.linked }

func (e Entry) SetCookie() string { return e.setCookie }

type SessionKeeper interface {
	Session(ctx context.Context, tokenHash TokenHash) (*Session, error)
	SaveSession(ctx context.Context, session *Session) error
}

func NewResumer(sessions SessionKeeper, lifetime Lifetime) *Resumer {
	return &Resumer{sessions: sessions, lifetime: lifetime.WithDefaults()}
}

type Resumer struct {
	sessions SessionKeeper
	lifetime Lifetime
}

func (r *Resumer) Resume(ctx context.Context, cookieHeader string, now time.Time) (Entry, error) {
	token, err := TokenFromCookies(cookieHeader)
	if err != nil {
		return Entry{}, fmt.Errorf("failed to read the cookie: %w", err)
	}

	session, err := r.sessions.Session(ctx, token.Hash())
	if err != nil {
		return Entry{}, fmt.Errorf("failed to find the session: %w", err)
	}
	if err := session.ExpiryError(now); err != nil {
		return Entry{}, fmt.Errorf("failed to resume the session: %w", err)
	}

	if !session.Extendable(now, r.lifetime) {
		return Entry{account: session.Account(), linked: session.Linked()}, nil
	}

	extended := session.Extended(now, r.lifetime)
	if err := r.sessions.SaveSession(ctx, extended); err != nil {
		return Entry{}, fmt.Errorf("failed to save the extended session: %w", err)
	}
	return Entry{account: extended.Account(), linked: extended.Linked(), setCookie: extended.Cookie(token, now)}, nil
}

// A cookie that holds no live session: none was sent, none is kept, or it is past its expiry.
func Ended(err error) bool {
	return errors.Is(err, ErrNoSessionCookie) ||
		errors.Is(err, ErrSessionNotFound) ||
		errors.Is(err, ErrSessionExpired)
}

type GuestCreator interface {
	CreateGuest(ctx context.Context, session *Session) error
}

func NewGuests(store GuestCreator, ids IDProvider, tokens TokenGenerator, lifetime Lifetime) *Guests {
	return &Guests{store: store, ids: ids, tokens: tokens, lifetime: lifetime.WithDefaults()}
}

type Guests struct {
	store    GuestCreator
	ids      IDProvider
	tokens   TokenGenerator
	lifetime Lifetime
}

func (g *Guests) Start(ctx context.Context, now time.Time) (Entry, error) {
	account, err := g.ids.NewID()
	if err != nil {
		return Entry{}, fmt.Errorf("failed to get an account id: %w", err)
	}
	token, err := g.tokens.NewToken()
	if err != nil {
		return Entry{}, fmt.Errorf("failed to get a session token: %w", err)
	}

	session := GuestSession(account, token, g.lifetime, now)
	if err := g.store.CreateGuest(ctx, session); err != nil {
		return Entry{}, fmt.Errorf("failed to store the guest: %w", err)
	}
	return Entry{account: session.Account(), setCookie: session.Cookie(token, now)}, nil
}
