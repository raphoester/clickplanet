package grants

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

type GrantID uuid.UUID

func (id GrantID) String() string {
	return uuid.UUID(id).String()
}

type AccountID = accounts.AccountID

var (
	ErrInvalidGrant  = errors.New("the grant is invalid")
	ErrGrantNotFound = errors.New("no such grant")
	ErrGrantExpired  = errors.New("the grant has expired")
)

func GrantIDOf(value string) (GrantID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return GrantID{}, fmt.Errorf("%w: %q is not a grant id", ErrGrantNotFound, value)
	}
	return GrantID(id), nil
}

// Long enough for the event bus to try a join again for four minutes, and short enough that a token at rest is soon
// gone.
const Lifetime = 10 * time.Minute

// The leave a Discord sign-in gave to add its user to the game's Discord server: the account it signed in to, the
// Discord user, and the access token that may join (scope guilds.join). Kept for Lifetime, for the alerts module to
// join with.
type Grant struct {
	id        GrantID
	account   AccountID
	user      string
	token     string
	expiresAt time.Time
}

func NewGrant(id GrantID, account AccountID, user string, token string, now time.Time) (Grant, error) {
	return Snapshot{ID: id, Account: account, User: user, Token: token, ExpiresAt: now.Add(Lifetime)}.Restore()
}

func (g Grant) ID() GrantID {
	return g.id
}

func (g Grant) ExpiryError(now time.Time) error {
	if !now.Before(g.expiresAt) {
		return fmt.Errorf("%w at %s", ErrGrantExpired, g.expiresAt.Format(time.RFC3339))
	}
	return nil
}

func (g Grant) TakeSnapshot() Snapshot {
	return Snapshot{ID: g.id, Account: g.account, User: g.user, Token: g.token, ExpiresAt: g.expiresAt}
}

type Snapshot struct {
	ID      GrantID
	Account AccountID
	// The Discord user's id.
	User string
	// Never logged, and sealed at rest.
	Token     string
	ExpiresAt time.Time
}

func (s Snapshot) Restore() (Grant, error) {
	switch {
	case s.ID == GrantID{}:
		return Grant{}, fmt.Errorf("%w: no id", ErrInvalidGrant)
	case s.Account == AccountID{}:
		return Grant{}, fmt.Errorf("%w: no account", ErrInvalidGrant)
	case s.User == "":
		return Grant{}, fmt.Errorf("%w: no Discord user", ErrInvalidGrant)
	case s.Token == "":
		return Grant{}, fmt.Errorf("%w: no token", ErrInvalidGrant)
	case s.ExpiresAt.IsZero():
		return Grant{}, fmt.Errorf("%w: no expiry", ErrInvalidGrant)
	}
	return Grant{id: s.ID, account: s.Account, user: s.User, token: s.Token, expiresAt: s.ExpiresAt}, nil
}

// A store keeps grants and finds them by id. It holds no rule: whether a grant still answers is Grant.ExpiryError.
type Store interface {
	SaveGrant(ctx context.Context, grant Grant) error
	// ErrGrantNotFound for a grant it does not keep.
	Grant(ctx context.Context, id GrantID) (Grant, error)
	// Deletes the grants expired by then, limit at most, and answers how many it deleted.
	PruneGrants(ctx context.Context, expiredBy time.Time, limit int) (int, error)
}

// Seals the token a store keeps, for that grant alone: a sealed token copied to another grant does not open.
type Sealer interface {
	Sealed(grant GrantID, token string) ([]byte, error)
	Opened(grant GrantID, sealed []byte) (string, error)
}

type IDProvider interface {
	NewID() (GrantID, error)
}
