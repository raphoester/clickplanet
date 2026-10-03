package marketingmessage

import (
	marketingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/marketing/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
)

func StateOf(state subscriptions.State) marketingv1.SubscriptionState {
	switch state {
	case subscriptions.StateWaiting:
		return marketingv1.SubscriptionState_SUBSCRIPTION_STATE_WAITING
	case subscriptions.StateActive:
		return marketingv1.SubscriptionState_SUBSCRIPTION_STATE_ACTIVE
	case subscriptions.StateNone, subscriptions.StateWithdrawn:
		return marketingv1.SubscriptionState_SUBSCRIPTION_STATE_NONE
	default:
		return marketingv1.SubscriptionState_SUBSCRIPTION_STATE_UNSPECIFIED
	}
}
