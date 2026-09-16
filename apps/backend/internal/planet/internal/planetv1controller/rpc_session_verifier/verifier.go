// Package rpc_session_verifier checks click tokens with a key the auth module
// hands over, asked for once over the internal listener and kept for the life of
// the process. This context never holds the seed, so it cannot mint.
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

// fetched asks auth once. The key does not change while the process runs, so the
// call happens on the first click after a boot and never again — a click is one
// signature check, as it was when this module read a key of its own.
//
// It cannot happen at boot instead: cpbootstrap builds every module before it
// listens, so the internal listener is not up while this one is being built. By
// the time a click arrives the server is serving, so in practice it is never late.
//
// A failed fetch is not remembered, so the next click tries again.
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

	// Its own deadline rather than the click's: the answer is the whole process's,
	// so one caller going away must not cancel the fetch every other one is waiting on.
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

	v.logger.Info("took the click token verifying key from the auth module")
	v.verifier = verifier

	return verifier, nil
}
