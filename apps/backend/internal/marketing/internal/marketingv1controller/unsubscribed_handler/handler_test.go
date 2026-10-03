package unsubscribed_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	marketingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/marketing/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/marketingv1controller/unsubscribed_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
)

type stubUseCase struct {
	err       error
	withdrawn []subscriptions.Address
}

func (s *stubUseCase) Execute(_ context.Context, address subscriptions.Address) error {
	s.withdrawn = append(s.withdrawn, address)
	return s.err
}

func unsubscribed(t *testing.T, useCase *stubUseCase, event, email string) error {
	t.Helper()

	_, err := unsubscribed_handler.New(useCase).Unsubscribed(t.Context(),
		connect.NewRequest(&marketingv1.UnsubscribedRequest{Event: event, Email: email}))
	return err //nolint:wrapcheck // the test reads the connect code.
}

func TestAnUnsubscribeWithdrawsTheAddress(t *testing.T) {
	for _, event := range []string{"unsubscribe", "unsubscribed"} {
		useCase := &stubUseCase{}

		require.NoError(t, unsubscribed(t, useCase, event, " Ada@Example.com"))

		assert.Equal(t, []subscriptions.Address{"ada@example.com"}, useCase.withdrawn, event)
	}
}

func TestAnyOtherEventIsAnsweredAndChangesNothing(t *testing.T) {
	useCase := &stubUseCase{}

	require.NoError(t, unsubscribed(t, useCase, "listAddition", "ada@example.com"))
	require.NoError(t, unsubscribed(t, useCase, "", "ada@example.com"))

	assert.Empty(t, useCase.withdrawn)
}

func TestAnAddressThatIsNotOneIsInvalidArgument(t *testing.T) {
	useCase := &stubUseCase{}

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(unsubscribed(t, useCase, "unsubscribe", "not an address")))
	assert.Empty(t, useCase.withdrawn)
}

func TestAFailureIsLeftToTheErrorNet(t *testing.T) {
	err := unsubscribed(t, &stubUseCase{err: errors.New("postgres is down")}, "unsubscribe", "ada@example.com")

	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}
