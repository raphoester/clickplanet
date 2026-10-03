package subscribe_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	marketingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/marketing/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/marketingv1controller/subscribe_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

const account = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

type stubUseCase struct {
	state   subscriptions.State
	err     error
	address string
}

func (s *stubUseCase) Execute(_ context.Context, _ subscriptions.AccountID, address string) (subscriptions.State, error) {
	s.address = address
	return s.state, s.err
}

func subscribe(t *testing.T, useCase *stubUseCase, address string) (*marketingv1.SubscribeResponse, error) {
	t.Helper()

	res, err := subscribe_handler.New(useCase).Subscribe(cpctx.AddAccountToContext(t.Context(), account),
		connect.NewRequest(&marketingv1.SubscribeRequest{Address: address}))
	if err != nil {
		return nil, fmt.Errorf("Subscribe failed: %w", err)
	}
	return res.Msg, nil
}

func TestTheStateTheSubscriptionTookIsAnswered(t *testing.T) {
	useCase := &stubUseCase{state: subscriptions.StateWaiting}

	res, err := subscribe(t, useCase, "ada@work.example")

	require.NoError(t, err)
	assert.Equal(t, marketingv1.SubscriptionState_SUBSCRIPTION_STATE_WAITING, res.GetState())
	assert.Equal(t, "ada@work.example", useCase.address)
}

func TestEachRefusalHasItsCode(t *testing.T) {
	for err, code := range map[error]connect.Code{
		fmt.Errorf("%w: no domain", subscriptions.ErrAddressInvalid): connect.CodeInvalidArgument,
		subscriptions.ErrNotLinked:                                   connect.CodePermissionDenied,
		subscriptions.ErrAlreadySubscribed:                           connect.CodeFailedPrecondition,
		fmt.Errorf("%w: 502", subscriptions.ErrAudienceUnreachable):  connect.CodeUnavailable,
		errors.New("postgres is down"):                               connect.CodeUnknown,
	} {
		t.Run(err.Error(), func(t *testing.T) {
			_, got := subscribe(t, &stubUseCase{err: err}, "ada@example.com")

			assert.Equal(t, code, connect.CodeOf(got))
		})
	}
}

func TestNoAccountIsUnauthenticated(t *testing.T) {
	_, err := subscribe_handler.New(&stubUseCase{}).Subscribe(t.Context(), connect.NewRequest(&marketingv1.SubscribeRequest{}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
