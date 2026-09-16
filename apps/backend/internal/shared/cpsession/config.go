package cpsession

import (
	"errors"
	"fmt"
	"time"
)

// Config is the token half of the `auth:` block, and it is shared because two
// bounded contexts read it: the auth context mints with it, the planet context
// verifies with it. Each builds its own Signer from the same settings, so the
// two never share an object and cannot drift — the same secret and TTL produce
// the same MAC.
type Config struct {
	// Off registers nothing: auth.v1 and session.v1 404, and clicks are judged
	// on address alone, as they were before this existed.
	Enabled bool

	// Off counts what enforcing would refuse without refusing it. Ship in this
	// mode, watch click_session_checks, then turn it on.
	Enforce bool

	// Signs the tokens. Required once Enabled: a server that invents one
	// cannot verify what another part of itself minted.
	Secret string

	// How long a minted token is accepted for.
	TTL time.Duration
}

const defaultTTL = time.Hour

// withDefaults fills an unset TTL only. A negative one is a typo rather than an
// omission, and NewSigner refuses it instead of quietly picking an hour.
func (c Config) withDefaults() Config {
	if c.TTL == 0 {
		c.TTL = defaultTTL
	}

	return c
}

// Validate is what keeps a signer from being built twice over different keys.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Secret == "" {
		return errors.New("auth.secret is empty while auth.enabled is true: set SESSION_SECRET")
	}
	if c.TTL < 0 {
		return fmt.Errorf("auth.ttl must be positive, got %s", c.TTL)
	}

	return nil
}
