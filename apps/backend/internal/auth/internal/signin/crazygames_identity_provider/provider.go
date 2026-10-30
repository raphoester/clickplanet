package crazygames_identity_provider

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	algorithm = "RS256"
	// The SDK's IN_GAME_AUTH key; the legacy one is not taken.
	keyID = "ingame-auth"
)

type Provider struct {
	gameID string
	keys   *KeySet
	clock  cptime.Clock
}

var _ signin.TokenProvider = (*Provider)(nil)

func New(gameID string, keys *KeySet, clock cptime.Clock) *Provider {
	return &Provider{gameID: gameID, keys: keys, clock: clock}
}

type header struct {
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"`
}

type claims struct {
	UserID    userID `json:"userId"`
	GameID    string `json:"gameId"`
	ExpiresAt int64  `json:"exp"`
}

type userID string

func (u *userID) UnmarshalJSON(raw []byte) error {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		*u = userID(text)
		return nil
	}
	var number json.Number
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&number); err != nil {
		return fmt.Errorf("the userId is neither a string nor a number: %w", err)
	}
	*u = userID(number.String())
	return nil
}

func (p *Provider) Claim(ctx context.Context, token string) (*accounts.Claim, error) {
	read, err := p.claimsOf(ctx, token)
	if errors.Is(err, errUnreachable) {
		return nil, fmt.Errorf("crazygames: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: crazygames: %w", signin.ErrProviderRefused, err)
	}
	claim := accounts.ClaimOf(string(read.UserID), "", false)
	return &claim, nil
}

func (p *Provider) claimsOf(ctx context.Context, token string) (*claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("the token has %d parts, want 3", len(parts))
	}
	var head header
	if err := decodePart(parts[0], &head); err != nil {
		return nil, fmt.Errorf("the token's header: %w", err)
	}
	if head.Algorithm != algorithm || head.KeyID != keyID {
		return nil, fmt.Errorf("the token is signed %s with key %q, want %s with %q", head.Algorithm, head.KeyID, algorithm, keyID)
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("the token's signature is not base64url: %w", err)
	}
	if err := p.verify(ctx, parts[0]+"."+parts[1], signature); err != nil {
		return nil, err
	}

	var read claims
	if err := decodePart(parts[1], &read); err != nil {
		return nil, fmt.Errorf("the token's payload: %w", err)
	}
	switch {
	case !p.clock.Now().Before(time.Unix(read.ExpiresAt, 0)):
		return nil, fmt.Errorf("the token expired at %d", read.ExpiresAt)
	case read.GameID != p.gameID:
		return nil, fmt.Errorf("the token is for the game %q", read.GameID)
	case read.UserID == "":
		return nil, errors.New("the token names no user")
	}
	return &read, nil
}

// A key CrazyGames rotated is fetched again once the cached one fails.
func (p *Provider) verify(ctx context.Context, signed string, signature []byte) error {
	digest := sha256.Sum256([]byte(signed))
	var verifyErr error
	for _, fresh := range []bool{false, true} {
		key, err := p.keys.key(ctx, keyID, fresh)
		if err != nil {
			return err
		}
		if verifyErr = rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature); verifyErr == nil {
			return nil
		}
	}
	return fmt.Errorf("the token's signature does not match: %w", verifyErr)
}

func decodePart(part string, into any) error {
	raw, err := base64.RawURLEncoding.DecodeString(part)
	if err != nil {
		return fmt.Errorf("not base64url: %w", err)
	}
	if err := json.Unmarshal(raw, into); err != nil {
		return fmt.Errorf("does not decode: %w", err)
	}
	return nil
}
