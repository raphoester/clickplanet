// Package session wires the mint: what a caller has to prove before it may
// click. It builds everything it needs from its own config, including the
// signer — the clicks context builds an identical one from the same `session:`
// block rather than being handed this one.
package session

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/session/v1/sessionv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/bootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ratelimit"
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

func NewModule(config Config) bootstrap.Module {
	return bootstrap.Module{
		Name:    moduleName,
		Enabled: config.Enabled,
		DiSequence: func(_ context.Context, props bootstrap.Props) error {
			return build(config.withDefaults(), props)
		},
	}
}

func build(config Config, props bootstrap.Props) error {
	signer, err := kernelsession.NewSigner(config.Config)
	if err != nil {
		return fmt.Errorf("failed to build the session signer: %w", err)
	}

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

// Config is the `session:` block. The token half is the kernel's, because the
// clicks context declares the same type to verify what this one mints.
type Config struct {
	kernelsession.Config `koanf:",squash"`

	// Per-IP throttle on minting. Minting costs a siteverify round trip, so it
	// needs its own budget rather than the click one.
	RateLimiter ratelimit.Config

	Turnstile turnstile.Config
}

const defaultTurnstileAction = "session"

func (c Config) withDefaults() Config {
	if c.Turnstile.Action == "" {
		c.Turnstile.Action = defaultTurnstileAction
	}

	return c
}
