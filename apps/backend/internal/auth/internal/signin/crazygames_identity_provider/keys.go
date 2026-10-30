package crazygames_identity_provider

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const ProductionKeys = "https://userportal.crazygames.com/.well-known/jwks.json"

// A fresh fetch at most this often: a flood of forged tokens costs CrazyGames one request a minute.
const refetchAfter = time.Minute

const maxKeySetBytes = 1 << 20

var errUnreachable = errors.New("the key set could not be fetched")

type KeySet struct {
	url   string
	http  *http.Client
	clock cptime.Clock

	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
}

func NewKeySet(url string, httpClient *http.Client, clock cptime.Clock) *KeySet {
	return &KeySet{url: url, http: httpClient, clock: clock}
}

type jwks struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	KeyType string `json:"kty"`
	KeyID   string `json:"kid"`
	N       string `json:"n"`
	E       string `json:"e"`
}

func (s *KeySet) key(ctx context.Context, kid string, fresh bool) (*rsa.PublicKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, known := s.keys[kid]
	due := s.keys == nil || s.clock.Now().Sub(s.fetchedAt) >= refetchAfter
	if (!known || fresh) && due {
		keys, err := s.fetch(ctx)
		if err != nil {
			return nil, err
		}
		s.keys, s.fetchedAt = keys, s.clock.Now()
		key, known = s.keys[kid]
	}
	if !known {
		return nil, fmt.Errorf("no key %q in the key set", kid)
	}
	return key, nil
}

func (s *KeySet) fetch(ctx context.Context) (map[string]*rsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errUnreachable, err)
	}
	res, err := s.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errUnreachable, err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: answered %d", errUnreachable, res.StatusCode)
	}

	var set jwks
	if err := json.NewDecoder(io.LimitReader(res.Body, maxKeySetBytes)).Decode(&set); err != nil {
		return nil, fmt.Errorf("%w: it does not decode: %w", errUnreachable, err)
	}
	keys := map[string]*rsa.PublicKey{}
	for _, each := range set.Keys {
		if each.KeyType != "RSA" {
			continue
		}
		key, err := publicKeyOf(each)
		if err != nil {
			return nil, fmt.Errorf("%w: the key %q: %w", errUnreachable, each.KeyID, err)
		}
		keys[each.KeyID] = key
	}
	return keys, nil
}

func publicKeyOf(key jwk) (*rsa.PublicKey, error) {
	modulus, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return nil, fmt.Errorf("its modulus is not base64url: %w", err)
	}
	exponent, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return nil, fmt.Errorf("its exponent is not base64url: %w", err)
	}
	e := new(big.Int).SetBytes(exponent)
	if !e.IsInt64() || e.Int64() < 3 || e.Int64() >= 1<<31 {
		return nil, fmt.Errorf("its exponent %s is out of range", e)
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(modulus), E: int(e.Int64())}, nil
}
