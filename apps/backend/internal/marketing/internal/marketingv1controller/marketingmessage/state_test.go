package marketingmessage_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	marketingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/marketing/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/marketingv1controller/marketingmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
)

func TestAWithdrawalReadsAsNoSubscription(t *testing.T) {
	for state, want := range map[subscriptions.State]marketingv1.SubscriptionState{
		subscriptions.StateNone:      marketingv1.SubscriptionState_SUBSCRIPTION_STATE_NONE,
		subscriptions.StateWaiting:   marketingv1.SubscriptionState_SUBSCRIPTION_STATE_WAITING,
		subscriptions.StateActive:    marketingv1.SubscriptionState_SUBSCRIPTION_STATE_ACTIVE,
		subscriptions.StateWithdrawn: marketingv1.SubscriptionState_SUBSCRIPTION_STATE_NONE,
	} {
		assert.Equal(t, want, marketingmessage.StateOf(state), state)
	}
}
