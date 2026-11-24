package shopv1controller

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/shop/v1/shopv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/shopv1controller/create_checkout_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/shopv1controller/list_my_renewals_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/shopv1controller/list_plans_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/shopv1controller/stop_renewal_handler"
)

type ShopService struct {
	list_plans_handler.ListPlansHandler
	create_checkout_handler.CreateCheckoutHandler
	list_my_renewals_handler.ListMyRenewalsHandler
	stop_renewal_handler.StopRenewalHandler
}

var _ shopv1connect.ShopServiceHandler = ShopService{}

var ErrNoSession = errors.New("a payment requires a session; call auth.v1.AuthService/CreateSession first")

// The buyer's own calls name its account; the plans are public.
func NewSessionInterceptor(
	verifier cpconnect.SessionVerifier, clock cptime.Clock, registerer prometheus.Registerer,
) connect.Interceptor {
	checks := promauto.With(registerer).NewCounterVec(prometheus.CounterOpts{
		Name: "shop_session_checks",
		Help: "Shop calls by the verdict on the session token they carried",
	}, []string{"verdict"})

	return cpconnect.NewSessionInterceptor(
		verifier,
		clock,
		ErrNoSession,
		func(verdict cpconnect.SessionVerdict) { checks.WithLabelValues(string(verdict)).Inc() },
		cpconnect.Identified(shopv1connect.ShopServiceCreateCheckoutProcedure),
		cpconnect.Identified(shopv1connect.ShopServiceListMyRenewalsProcedure),
		cpconnect.Identified(shopv1connect.ShopServiceStopRenewalProcedure),
	)
}
