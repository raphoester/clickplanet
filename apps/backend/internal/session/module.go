// Package session wires the mint: what a caller has to prove before it may
// click. Everything below it — the domain, the attesters, the edge — is what
// this file assembles; the signer it mints with is built by the composition
// root, because the clicks context verifies with the same one.
package session

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/session/v1/sessionv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/bootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/secrets"
	kernelsession "github.com/raphoester/clickplanet.lol-backend/internal/kernel/session"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/turnstile"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/xtime"
	"github.com/raphoester/clickplanet.lol-backend/internal/session/adapters/primary/http/sessionv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/session/adapters/secondary/open_attester"
	"github.com/raphoester/clickplanet.lol-backend/internal/session/adapters/secondary/turnstile_attester"
	"github.com/raphoester/clickplanet.lol-backend/internal/session/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/session/domain/session_service"
)

const moduleName = "session"

// NewSigner builds the thing this context mints with and the clicks context
// verifies with.
//
// It is deliberately not built inside the DI sequence: it is the one dependency
// two contexts share, so the composition root holds it and hands it to both.
// Neither context can then reach into the other for it, and a build with
// sessions off is a nil signer rather than a module that half exists.
func NewSigner(config Config, logger logging.Logger) (*kernelsession.Signer, error) {
	config = config.withDefaults()

	secret := config.Secret
	if secret == "" {
		generated, err := secrets.RandomHex()
		if err != nil {
			return nil, fmt.Errorf("failed to generate a session secret: %w", err)
		}
		secret = generated
		logger.Warning("no session.secret configured, generated a random one: every session in flight is invalidated on each restart")
	}

	signer, err := kernelsession.NewSigner(secret, config.TTL)
	if err != nil {
		return nil, fmt.Errorf("failed to build the session signer: %w", err)
	}

	return signer, nil
}

func NewModule(config Config, signer *kernelsession.Signer) bootstrap.Module {
	return bootstrap.Module{
		Name: moduleName,
		DiSequence: func(_ context.Context, props bootstrap.Props) error {
			return build(config.withDefaults(), signer, props)
		},
	}
}

func build(config Config, signer *kernelsession.Signer, props bootstrap.Props) error {
	attester, err := newAttester(config, props.Logger)
	if err != nil {
		return err
	}

	mintLimiter := ratelimit.New(config.RateLimiter, xtime.ActualProvider{})
	props.Runners.Add("mint-limiter", mintLimiter.Run)

	err = props.RPC.Mount(sessionv1connect.NewSessionServiceHandler(
		sessionv1controller.NewSessionService(
			session_service.New(attester, signer, xtime.ActualProvider{}),
		),
		connect.WithInterceptors(
			sessionv1controller.NewErrorInterceptor(props.Logger),
			sessionv1controller.NewRateLimitInterceptor(mintLimiter),
		),
	))
	if err != nil {
		return err
	}

	props.Logger.Info("sessions enabled",
		lf.Any("ttl", config.TTL),
		lf.Bool("enforce", config.Enforce),
		lf.Bool("turnstile", config.Turnstile.Enabled),
	)

	return nil
}

func newAttester(config Config, logger logging.Logger) (domain.Attester, error) {
	if !config.Turnstile.Enabled {
		logger.Warning("session.turnstile is disabled: a session is minted for anyone who asks for one")
		return open_attester.New(), nil
	}

	attester, err := turnstile_attester.New(config.Turnstile)
	if err != nil {
		return nil, fmt.Errorf("failed to build the turnstile attester: %w", err)
	}

	return attester, nil
}

// Config gates the Click RPC on a token this server minted, which is the one
// thing an address-based defence cannot do: refuse a caller that never proved
// anything, however many addresses it has.
type Config struct {
	// Off registers nothing: session.v1.SessionService/ 404s and clicks are
	// judged on address alone, as they were before this existed.
	Enabled bool

	// Off counts what enforcing would refuse without refusing it. Ship in this
	// mode, watch click_session_checks, then turn it on.
	Enforce bool

	// Signs the tokens. Empty regenerates one at boot, which invalidates every
	// session in flight on each restart.
	Secret string

	// How long a minted token is accepted for.
	TTL time.Duration

	// Per-IP throttle on minting. Minting costs a siteverify round trip, so it
	// needs its own budget rather than the click one.
	RateLimiter ratelimit.Config

	Turnstile turnstile.Config
}

const (
	defaultTTL             = time.Hour
	defaultTurnstileAction = "session"
)

func (c Config) withDefaults() Config {
	if c.TTL <= 0 {
		c.TTL = defaultTTL
	}
	if c.Turnstile.Action == "" {
		c.Turnstile.Action = defaultTurnstileAction
	}

	return c
}
