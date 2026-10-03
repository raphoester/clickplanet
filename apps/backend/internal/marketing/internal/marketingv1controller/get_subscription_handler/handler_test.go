package get_subscription_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	marketingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/marketing/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/marketingv1controller/get_subscription_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

const account = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

type stubUseCase struct {
	status subscriptions.Status
	err    error
	asked  []subscriptions.AccountID
}

func (s *stubUseCase) Execute(_ context.Context, account subscriptions.AccountID) (subscriptions.Status, error) {
	s.asked = append(s.asked, account)
	return s.status, s.err
}

func TestTheStatusIsAnsweredForTheCaller(t *testing.T) {
	useCase := &stubUseCase{status: subscriptions.Status{State: subscriptions.StateWaiting, Address: "ada@work.example"}}

	res, err := get_subscription_handler.New(useCase).GetSubscription(cpctx.AddAccountToContext(t.Context(), account),
		connect.NewRequest(&marketingv1.GetSubscriptionRequest{}))

	require.NoError(t, err)
	assert.Equal(t, marketingv1.SubscriptionState_SUBSCRIPTION_STATE_WAITING, res.Msg.GetState())
	assert.Equal(t, "ada@work.example", res.Msg.GetAddress())
	require.Len(t, useCase.asked, 1)
	assert.Equal(t, account, useCase.asked[0].String())
}

func TestNoAccountIsUnauthenticatedAndAsksNobody(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := get_subscription_handler.New(useCase).GetSubscription(t.Context(), connect.NewRequest(&marketingv1.GetSubscriptionRequest{}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.Empty(t, useCase.asked)
}

func TestAFailureIsLeftToTheErrorNet(t *testing.T) {
	_, err := get_subscription_handler.New(&stubUseCase{err: errors.New("postgres is down")}).
		GetSubscription(cpctx.AddAccountToContext(t.Context(), account), connect.NewRequest(&marketingv1.GetSubscriptionRequest{}))

	require.Error(t, err)
	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}
