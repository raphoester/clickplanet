package cpsession

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// SignerConfig is the minting half of the `auth:` block: only the auth module declares it, so only auth holds the seed.
type SignerConfig struct {
	// Off registers nothing: auth.v1 and session.v1 404, and clicks are judged on address alone.
	Enabled bool

	// The Ed25519 seed, 32 bytes as 64 hex characters. Belongs in the environment.
	Secret string

	TTL time.Duration
}

// VerifierConfig is the checking half, and all the planet context gets: no seed, so it cannot mint.
type VerifierConfig struct {
	Enabled bool

	// Off counts what enforcing would refuse without refusing it. Ship in this mode.
	Enforce bool

	// Not a key in the file: the composition root derives it from the seed, so
	// there is one key to set and no second one to drift from it.
	PublicKey string
}

const defaultTTL = time.Hour

// withDefaults fills an unset TTL only; a negative one is a typo, and Validate refuses it.
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

// PublicKeyOf is the half a verifier needs, from the half that mints. The
// composition root calls it: planet's block is wired from auth's, never written twice.
func PublicKeyOf(secret string) (string, error) {
	key, err := parseSeed(secret)
	if err != nil {
		return "", err
	}

	public, ok := key.Public().(ed25519.PublicKey)
	if !ok {
		return "", errors.New("auth.secret did not yield an ed25519 public key")
	}

	return hex.EncodeToString(public), nil
}

// parseSeed names the variable an operator sets, not the field, since that is what they are reading.
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
		return nil, errors.New("the verifying key is empty: the composition root derives it from auth.secret")
	}

	key, err := hex.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("auth.publicKey is not hex: %w", err)
	}
	if len(key) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("auth.publicKey must be %d bytes as %d hex characters, got %d",
			ed25519.PublicKeySize, ed25519.PublicKeySize*2, len(key))
	}

	return key, nil
}
