package cpsession

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

type SignerConfig struct {
	Enabled bool

	Secret string

	TTL time.Duration
}

type VerifierConfig struct {
	Enabled bool

	Enforce bool
}

const defaultTTL = time.Hour

func (c SignerConfig) withDefaults() SignerConfig {
	if c.TTL == 0 {
		c.TTL = defaultTTL
	}

	return c
}

func (c SignerConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.TTL < 0 {
		return fmt.Errorf("auth.ttl must be positive, got %s", c.TTL)
	}

	_, err := parseSeed(c.Secret)

	return err
}

func parseSeed(value string) (ed25519.PrivateKey, error) {
	if value == "" {
		return nil, errors.New("auth.secret is empty while auth.enabled is true: set SESSION_SECRET (openssl rand -hex 32)")
	}

	seed, err := hex.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("auth.secret is not hex: %w", err)
	}
	if len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("auth.secret must be %d bytes as %d hex characters, got %d: openssl rand -hex %d",
			ed25519.SeedSize, ed25519.SeedSize*2, len(seed), ed25519.SeedSize)
	}

	return ed25519.NewKeyFromSeed(seed), nil
}

func parsePublicKey(value string) (ed25519.PublicKey, error) {
	if value == "" {
		return nil, errors.New("the verifying key is empty: auth.v1.InternalService answered nothing")
	}

	key, err := hex.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("the verifying key auth answered is not hex: %w", err)
	}
	if len(key) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("the verifying key auth answered must be %d bytes as %d hex characters, got %d",
			ed25519.PublicKeySize, ed25519.PublicKeySize*2, len(key))
	}

	return key, nil
}
