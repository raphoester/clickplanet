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
	pending  *pending
}

type pending struct {
	done     chan struct{}
	verifier *cpsession.Verifier
	err      error
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
	v.mu.Lock()
	if v.verifier != nil {
		defer v.mu.Unlock()
		return v.verifier, nil
	}
	current := v.pending
	if current == nil {
		current = &pending{done: make(chan struct{})}
		v.pending = current
		// Without the caller's cancellation: one caller leaving must not fail the others waiting on this fetch.
		go v.fetchKey(context.WithoutCancel(ctx), current)
	}
	v.mu.Unlock()

	select {
	case <-current.done:
		return current.verifier, current.err
	case <-ctx.Done():
		return nil, fmt.Errorf("stopped waiting for the click token verifying key: %w", ctx.Err())
	}
}

func (v *Verifier) fetchKey(ctx context.Context, current *pending) {
	current.verifier, current.err = v.key(ctx)

	v.mu.Lock()
	if current.err == nil {
		v.verifier = current.verifier
	}
	v.pending = nil
	v.mu.Unlock()

	close(current.done)
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

	v.logger.Info("the player module took the click token verifying key from the auth module")

	return verifier, nil
}
