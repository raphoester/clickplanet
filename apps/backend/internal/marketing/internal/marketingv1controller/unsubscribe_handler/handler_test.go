package unsubscribe_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	marketingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/marketing/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/marketingv1controller/unsubscribe_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

const account = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

type stubUseCase struct {
	err    error
	called bool
}

func (s *stubUseCase) Execute(context.Context, subscriptions.AccountID) error {
	s.called = true
	return s.err
}

func unsubscribe(ctx context.Context, useCase *stubUseCase) error {
	_, err := unsubscribe_handler.New(useCase).Unsubscribe(ctx, connect.NewRequest(&marketingv1.UnsubscribeRequest{}))
	return err //nolint:wrapcheck // the test reads the connect code.
}

func TestTheCallerIsUnsubscribed(t *testing.T) {
	useCase := &stubUseCase{}

	require.NoError(t, unsubscribe(cpctx.AddAccountToContext(t.Context(), account), useCase))

	assert.True(t, useCase.called)
}

func TestAListThatCannotBeReachedIsUnavailable(t *testing.T) {
	err := unsubscribe(cpctx.AddAccountToContext(t.Context(), account),
		&stubUseCase{err: fmt.Errorf("%w: timeout", subscriptions.ErrAudienceUnreachable)})

	assert.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}

func TestAnyOtherFailureIsLeftToTheErrorNet(t *testing.T) {
	err := unsubscribe(cpctx.AddAccountToContext(t.Context(), account), &stubUseCase{err: errors.New("postgres is down")})

	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}

func TestNoAccountIsUnauthenticated(t *testing.T) {
	useCase := &stubUseCase{}

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(unsubscribe(t.Context(), useCase)))
	assert.False(t, useCase.called)
}
