package cpsessionverifier

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/sync/singleflight"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type Dialer interface {
	Dial() (connect.HTTPClient, string, error)
}

const fetchTimeout = 2 * time.Second

type Verifier struct {
	dial   Dialer
	logger *slog.Logger

	fetches  singleflight.Group
	verifier atomic.Pointer[cpsession.Verifier]
}

func New(dial Dialer, logger *slog.Logger) *Verifier {
	return &Verifier{dial: dial, logger: logger}
}

func (v *Verifier) Verify(ctx context.Context, token string, ip string, now time.Time) (*cpsession.Claims, error) {
	verifier, err := v.fetched(ctx)
	if err != nil {
		return nil, err
	}

	return verifier.Verify(token, ip, now)
}

func (v *Verifier) fetched(ctx context.Context) (*cpsession.Verifier, error) {
	if verifier := v.verifier.Load(); verifier != nil {
		return verifier, nil
	}

	fetch := v.fetches.DoChan("key", func() (any, error) {
		if verifier := v.verifier.Load(); verifier != nil {
			return verifier, nil
		}
		// Without the caller's cancellation: one caller leaving must not fail the others waiting on this fetch.
		verifier, err := v.key(context.WithoutCancel(ctx))
		if err != nil {
			return nil, err
		}
		v.verifier.Store(verifier)
		return verifier, nil
	})

	select {
	case result := <-fetch:
		if result.Err != nil {
			return nil, result.Err
		}
		return result.Val.(*cpsession.Verifier), nil
	case <-ctx.Done():
		return nil, fmt.Errorf("stopped waiting for the click token verifying key: %w", ctx.Err())
	}
}

func (v *Verifier) key(ctx context.Context) (*cpsession.Verifier, error) {
	client, baseURL, err := v.dial.Dial()
	if err != nil {
		return nil, fmt.Errorf("failed to reach the auth module: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	res, err := authv1connect.NewInternalServiceClient(client, baseURL).
		GetVerifyingKey(ctx, connect.NewRequest(&authv1.GetVerifyingKeyRequest{}))
	if err != nil {
		return nil, fmt.Errorf("failed to ask auth for the click token verifying key: %w", err)
	}

	verifier, err := cpsession.NewVerifier(res.Msg.GetPublicKey())
	if err != nil {
		return nil, fmt.Errorf("auth answered a verifying key this server cannot use: %w", err)
	}

	v.logger.Info("took the click token verifying key from the auth module")

	return verifier, nil
}
