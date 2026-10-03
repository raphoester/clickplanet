package marketingv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/marketing/v1/marketingv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/marketingv1controller/get_subscription_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/marketingv1controller/subscribe_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/marketingv1controller/unsubscribe_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/marketingv1controller/unsubscribed_handler"
)

type SubscriptionService struct {
	get_subscription_handler.GetSubscriptionHandler
	subscribe_handler.SubscribeHandler
	unsubscribe_handler.UnsubscribeHandler
}

var _ marketingv1connect.SubscriptionServiceHandler = SubscriptionService{}

type BrevoService struct {
	unsubscribed_handler.UnsubscribedHandler
}

var _ marketingv1connect.BrevoServiceHandler = BrevoService{}
