package admintokens

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

const (
	AccessPrefix  = "cpat_"
	RefreshPrefix = "cprt_"

	AccessLifetime  = time.Hour
	RefreshLifetime = 90 * 24 * time.Hour
)

var ErrUnknownToken = errors.New("no live refresh token has this hash")

type Kind string

const (
	KindAccess  Kind = "access"
	KindRefresh Kind = "refresh"
)

type Secret string

func (s Secret) Hash() accounts.TokenHash {
	sum := sha256.Sum256([]byte(s))
	return sum[:]
}

type SecretGenerator interface {
	NewSecret() (string, error)
}

type ClientID string

type Holder struct {
	Account  accounts.AccountID
	ClientID ClientID
	Client   string
}

type Snapshot struct {
	Hash      accounts.TokenHash
	Kind      Kind
	Holder    Holder
	CreatedAt time.Time
	ExpiresAt time.Time
}

type Pair struct {
	access    Secret
	refresh   Secret
	holder    Holder
	createdAt time.Time
}

type PairSnapshot struct {
	Access  Secret
	Refresh Secret
}

func (p *Pair) TakeSnapshot() PairSnapshot {
	return PairSnapshot{Access: p.access, Refresh: p.refresh}
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
	access, err := i.secrets.NewSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to draw an access token: %w", err)
	}
	refresh, err := i.secrets.NewSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to draw a refresh token: %w", err)
	}

	return &Pair{
		access: Secret(AccessPrefix + access), refresh: Secret(RefreshPrefix + refresh), holder: holder, createdAt: now,
	}, nil
}
