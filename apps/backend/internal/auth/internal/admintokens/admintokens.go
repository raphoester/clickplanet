package admintokens

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

const (
	AccessPrefix  = "cpat_"
	RefreshPrefix = "cprt_"

	AccessLifetime  = time.Hour
	RefreshLifetime = 90 * 24 * time.Hour
)

var (
	ErrUnknownToken = errors.New("no live refresh token has this hash")
	ErrReused       = fmt.Errorf("%w: traded already, so its connection is ended", ErrUnknownToken)
	ErrInvalidToken = errors.New("a token this server does not keep")
)

type Kind string

const (
	KindAccess  Kind = "access"
	KindRefresh Kind = "refresh"
	KindSpent   Kind = "spent"
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

type ConnectionID string

type Holder struct {
	Account  accounts.AccountID
	ClientID ClientID
	Client   string
}

type Snapshot struct {
	Hash       accounts.TokenHash
	Kind       Kind
	Holder     Holder
	Connection ConnectionID
	CreatedAt  time.Time
	ExpiresAt  time.Time
}

func (s Snapshot) Restore() (*Token, error) {
	if len(s.Hash) == 0 || s.Connection == "" || !slices.Contains([]Kind{KindAccess, KindRefresh, KindSpent}, s.Kind) {
		return nil, fmt.Errorf("%w: kind %q, connection %q", ErrInvalidToken, s.Kind, s.Connection)
	}
	return &Token{snapshot: s}, nil
}

type Token struct {
	snapshot Snapshot
}

func (t *Token) TakeSnapshot() Snapshot {
	return t.snapshot
}

type Pair struct {
	access     Secret
	refresh    Secret
	holder     Holder
	connection ConnectionID
	createdAt  time.Time
}

type PairSnapshot struct {
	Access  Secret
	Refresh Secret
}

func (p *Pair) TakeSnapshot() PairSnapshot {
	return PairSnapshot{Access: p.access, Refresh: p.refresh}
}

func (p *Pair) Tokens() []Snapshot {
	token := Snapshot{Holder: p.holder, Connection: p.connection, CreatedAt: p.createdAt}
	access, refresh := token, token
	access.Hash, access.Kind, access.ExpiresAt = p.access.Hash(), KindAccess, p.createdAt.Add(AccessLifetime)
	refresh.Hash, refresh.Kind, refresh.ExpiresAt = p.refresh.Hash(), KindRefresh, p.createdAt.Add(RefreshLifetime)
	return []Snapshot{access, refresh}
}

type Trade struct {
	Kept    []Snapshot
	Pair    *Pair
	Refusal error
}

func NewIssuer(secrets SecretGenerator) *Issuer {
	return &Issuer{secrets: secrets}
}

type Issuer struct {
	secrets SecretGenerator
}

func (i *Issuer) Connected(holder Holder, now time.Time) (*Pair, error) {
	connection, err := i.secrets.NewSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to draw a connection id: %w", err)
	}
	return i.pair(holder, ConnectionID(connection), now)
}

// A trade spends the refresh token; one traded already ends every token of its connection, since a thief holds a copy.
func (i *Issuer) Trade(connection []*Token, traded Secret, client ClientID, now time.Time) (Trade, error) {
	found := slices.IndexFunc(connection, func(token *Token) bool { return slices.Equal(token.snapshot.Hash, traded.Hash()) })
	if found < 0 {
		return Trade{Refusal: ErrUnknownToken}, nil
	}

	held := connection[found].snapshot
	switch {
	case held.Kind == KindSpent:
		return Trade{Kept: ended(connection, now), Refusal: ErrReused}, nil
	case held.Kind != KindRefresh || !now.Before(held.ExpiresAt):
		return Trade{Refusal: fmt.Errorf("%w: not a live refresh token", ErrUnknownToken)}, nil
	}

	spent := held
	spent.Kind = KindSpent
	if held.Holder.ClientID != client {
		return Trade{Kept: []Snapshot{spent}, Refusal: fmt.Errorf("%w: issued to another client", ErrUnknownToken)}, nil
	}

	pair, err := i.pair(held.Holder, held.Connection, now)
	if err != nil {
		return Trade{}, err
	}
	return Trade{Kept: append([]Snapshot{spent}, pair.Tokens()...), Pair: pair}, nil
}

func ended(connection []*Token, now time.Time) []Snapshot {
	kept := make([]Snapshot, 0, len(connection))
	for _, token := range connection {
		end := token.snapshot
		if now.Before(end.ExpiresAt) {
			end.ExpiresAt = now
		}
		kept = append(kept, end)
	}
	return kept
}

func (i *Issuer) pair(holder Holder, connection ConnectionID, now time.Time) (*Pair, error) {
	access, err := i.secrets.NewSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to draw an access token: %w", err)
	}
	refresh, err := i.secrets.NewSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to draw a refresh token: %w", err)
	}

	return &Pair{
		access: Secret(AccessPrefix + access), refresh: Secret(RefreshPrefix + refresh), holder: holder,
		connection: connection, createdAt: now,
	}, nil
}
