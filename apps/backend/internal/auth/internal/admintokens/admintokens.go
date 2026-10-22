package admintokens

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

const (
	// What each secret starts with, so a person, a log and a secret scanner can tell one from any other string.
	AccessPrefix  = "cpat_"
	RefreshPrefix = "cprt_"

	// An access token is short, and the app trades its refresh token for a new pair when it lapses. A refresh token is
	// good once: each trade hands a new one, so a connection used once a season never lapses.
	AccessLifetime  = time.Hour
	RefreshLifetime = 90 * 24 * time.Hour
)

var ErrUnknownToken = errors.New("no live refresh token has this hash")

// An access token reaches the admin listener; a refresh token only buys the next pair.
type Kind string

const (
	KindAccess  Kind = "access"
	KindRefresh Kind = "refresh"
)

// What the app holds. The server keeps its hash and nothing else.
type Secret string

func (s Secret) Hash() accounts.TokenHash {
	sum := sha256.Sum256([]byte(s))
	return sum[:]
}

type SecretGenerator interface {
	NewSecret(prefix string) (Secret, error)
}

// The app a token is issued to: its OAuth client id, and the name it gave itself, which the logs show.
type Holder struct {
	Account  accounts.AccountID
	ClientID string
	Client   string
}

type Snapshot struct {
	Hash      accounts.TokenHash
	Kind      Kind
	Holder    Holder
	CreatedAt time.Time
	ExpiresAt time.Time
}

// An access token and the refresh token that replaces it, issued together.
type Pair struct {
	access    Secret
	refresh   Secret
	holder    Holder
	createdAt time.Time
}

func (p *Pair) Access() Secret {
	return p.access
}

func (p *Pair) Refresh() Secret {
	return p.refresh
}

func (p *Pair) TakeSnapshots() []Snapshot {
	return []Snapshot{
		{Hash: p.access.Hash(), Kind: KindAccess, Holder: p.holder, CreatedAt: p.createdAt, ExpiresAt: p.createdAt.Add(AccessLifetime)},
		{Hash: p.refresh.Hash(), Kind: KindRefresh, Holder: p.holder, CreatedAt: p.createdAt, ExpiresAt: p.createdAt.Add(RefreshLifetime)},
	}
}

func NewIssuer(secrets SecretGenerator) *Issuer {
	return &Issuer{secrets: secrets}
}

type Issuer struct {
	secrets SecretGenerator
}

func (i *Issuer) Pair(holder Holder, now time.Time) (*Pair, error) {
	access, err := i.secrets.NewSecret(AccessPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to draw an access token: %w", err)
	}
	refresh, err := i.secrets.NewSecret(RefreshPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to draw a refresh token: %w", err)
	}

	return &Pair{access: access, refresh: refresh, holder: holder, createdAt: now}, nil
}

// Who an access token names is a query of the internal service: the store only writes, and spends a refresh token.
type Store interface {
	SaveTokens(ctx context.Context, tokens []Snapshot) error
	// A live refresh token's holder, deleted in the same statement, so it is spent once. ErrUnknownToken otherwise.
	SpendRefresh(ctx context.Context, hash accounts.TokenHash, now time.Time) (Holder, error)
	// A lapsed token is refused anyway: this keeps the table to the live ones.
	DeleteLapsed(ctx context.Context, now time.Time) error
}
