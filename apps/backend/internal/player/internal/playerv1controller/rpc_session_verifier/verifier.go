package rpc_session_verifier

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"connectrpc.com/connect"

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

	mu       sync.Mutex
	verifier *cpsession.Verifier
}

func New(dial Dialer, logger *slog.Logger) *Verifier {
	return &Verifier{dial: dial, logger: logger}
}

func (v *Verifier) Verify(token string, ip string, now time.Time) (*cpsession.Claims, error) {
	verifier, err := v.fetched()
	if err != nil {
		return nil, err
	}

	return verifier.Verify(token, ip, now)
}

func (v *Verifier) fetched() (*cpsession.Verifier, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.verifier != nil {
		return v.verifier, nil
	}

	client, baseURL, err := v.dial.Dial()
	if err != nil {
		return nil, fmt.Errorf("failed to reach the auth module: %w", err)
	}

	// Not the caller's ctx: one caller leaving must not cancel the fetch others wait on.
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
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

	v.logger.Info("the player module took the click token verifying key from the auth module")
	v.verifier = verifier

	return verifier, nil
}
