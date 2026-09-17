// Package rpc_session_verifier checks click tokens with a key the auth module hands over, asked for once
// over the internal listener and kept for the life of the process. This module never holds the seed.
//
// It is planet's verifier, copied, as player's is: a module cannot import another's interior, and the one thing
// they share, auth.v1.InternalService, is already a contract.
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

// Dialer is cpbootstrap's internal listener: the only way one module reaches another.
type Dialer interface {
	Dial() (connect.HTTPClient, string, error)
}

// fetchTimeout bounds the one call. It is loopback, so this is a stuck socket rather than a slow network.
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

// fetched asks auth once, on the first call after a boot: the key does not change while the process runs.
// It cannot ask at boot, since the internal listener is not up while modules are built. A failed fetch is
// not remembered, so the next call tries again.
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

	// Its own deadline rather than the call's: one caller going away must not cancel the fetch the others wait on.
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

	v.logger.Info("the chat module took the click token verifying key from the auth module")
	v.verifier = verifier

	return verifier, nil
}
