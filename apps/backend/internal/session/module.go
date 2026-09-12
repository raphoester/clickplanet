// Package session wires the mint: what a caller has to prove before it may
// click. It builds everything it needs from its own config, including the
// signer — the planet context builds an identical one from the same `session:`
// block rather than being handed this one.
package session

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/session/v1/sessionv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/session/internal/adapters/primary/http/sessionv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/session/internal/adapters/secondary/open_attester"
	"github.com/raphoester/clickplanet.lol-backend/internal/session/internal/adapters/secondary/turnstile_attester"
	"github.com/raphoester/clickplanet.lol-backend/internal/session/internal/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/session/internal/domain/session_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/session/internal/turnstile"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "session"

func NewModule(config Config) cpbootstrap.Module {
	return cpbootstrap.Module{
		Name:    moduleName,
		Enabled: config.Enabled,
		DiSequence: func(_ context.Context, props cpbootstrap.Props) error {
			return build(config.withDefaults(), props)
		},
	}
}

func build(config Config, props cpbootstrap.Props) error {
	signer, err := cpsession.NewSigner(config.Config)
	if err != nil {
		return fmt.Errorf("failed to build the session signer: %w", err)
	}

	attester, err := newAttester(config, props.Logger)
	if err != nil {
		return err
	}

	mintLimiter := cpratelimit.New(config.RateLimiter, cptime.SystemClock{})
	props.Runners.Add("mint-limiter", mintLimiter.Run)

	sessionService := sessionv1controller.NewSessionService(
		session_service.New(attester, signer, cptime.SystemClock{}),
		props.Logger,
	)

	err = props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return sessionv1connect.NewSessionServiceHandler(sessionService, options...)
	},
		sessionv1controller.NewRateLimitInterceptor(mintLimiter),
	)
	if err != nil {
		return err
	}

	props.Logger.Info("sessions enabled",
		slog.Duration("ttl", config.TTL),
		slog.Bool("enforce", config.Enforce),
		slog.Bool("turnstile", config.Turnstile.Enabled),
	)

	return nil
}

func newAttester(config Config, logger *slog.Logger) (domain.Attester, error) {
	if !config.Turnstile.Enabled {
		logger.Warn("session.turnstile is disabled: a session is minted for anyone who asks for one")
		return open_attester.New(), nil
	}

	attester, err := turnstile_attester.New(config.Turnstile)
	if err != nil {
		return nil, fmt.Errorf("failed to build the turnstile attester: %w", err)
	}

	return attester, nil
}

// Config is the `session:` block. The token half is the shared layer's, because the
// planet context declares the same type to verify what this one mints.
type Config struct {
	cpsession.Config `koanf:",squash"`

	// Per-IP throttle on minting. Minting costs a siteverify round trip, so it
	// needs its own budget rather than the click one.
	RateLimiter cpratelimit.Config

	Turnstile turnstile.Config
}

const defaultTurnstileAction = "session"

func (c Config) withDefaults() Config {
	if c.Turnstile.Action == "" {
		c.Turnstile.Action = defaultTurnstileAction
	}

	return c
}

// Validate checks the `session:` block, which the planet context also reads and
// therefore does not check itself.
func (c Config) Validate() error {
	return c.Config.Validate()
}
