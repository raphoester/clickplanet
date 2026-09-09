package app

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/session/v1/sessionv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/session"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/xtime"
	"github.com/raphoester/clickplanet.lol-backend/internal/session/adapters/primary/http/sessionv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/session/adapters/secondary/open_attester"
	"github.com/raphoester/clickplanet.lol-backend/internal/session/adapters/secondary/turnstile_attester"
	"github.com/raphoester/clickplanet.lol-backend/internal/session/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/session/domain/session_service"
)

// configureSessionIfEnabled runs before the clicks context, which reads the
// signer it leaves on the App to verify what this one minted. The two never
// meet anywhere else: minting is this context's business, and clicks only ever
// checks a signature.
func (a *App) configureSessionIfEnabled(_ context.Context) error {
	if !a.config.Session.Enabled {
		return nil
	}

	config := a.config.Session.withDefaults()

	secret := config.Secret
	if secret == "" {
		generated, err := randomSalt()
		if err != nil {
			return fmt.Errorf("failed to generate a session secret: %w", err)
		}
		secret = generated
		a.logger.Warning("no session.secret configured, generated a random one: every session in flight is invalidated on each restart")
	}

	signer, err := session.NewSigner(secret, config.TTL)
	if err != nil {
		return fmt.Errorf("failed to build the session signer: %w", err)
	}
	a.sessionSigner = signer

	attester, err := a.sessionAttester(config)
	if err != nil {
		return err
	}

	mintLimiter := ratelimit.New(config.RateLimiter, xtime.ActualProvider{})
	a.runners = append(a.runners, func() { mintLimiter.Run(a.ctx) })

	a.mountRPC(sessionv1connect.NewSessionServiceHandler(
		sessionv1controller.NewSessionService(
			session_service.New(attester, signer, xtime.ActualProvider{}),
		),
		connect.WithInterceptors(
			sessionv1controller.NewErrorInterceptor(a.logger),
			sessionv1controller.NewRateLimitInterceptor(mintLimiter),
		),
	))

	a.logger.Info("sessions enabled",
		lf.Any("ttl", config.TTL),
		lf.Bool("enforce", config.Enforce),
		lf.Bool("turnstile", config.Turnstile.Enabled),
	)

	return nil
}

func (a *App) sessionAttester(config SessionConfig) (domain.Attester, error) {
	if !config.Turnstile.Enabled {
		a.logger.Warning("session.turnstile is disabled: a session is minted for anyone who asks for one")
		return open_attester.New(), nil
	}

	attester, err := turnstile_attester.New(config.Turnstile)
	if err != nil {
		return nil, fmt.Errorf("failed to build the turnstile attester: %w", err)
	}

	return attester, nil
}
