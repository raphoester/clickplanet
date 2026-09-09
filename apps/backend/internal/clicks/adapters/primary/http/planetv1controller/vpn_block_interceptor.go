package planetv1controller

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ctxutil"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ipblock"
)

// ErrVPNBlocked is what the player is told. The message matters: the web app
// keys the dialog on the code, but this is the string that ends up in a curl
// and in anyone's console.
var ErrVPNBlocked = errors.New("clicks from VPN and proxy addresses are refused; turn yours off to play")

type ClickBlocklist interface {
	Blocked(ip string) (ipblock.List, bool)
}

// NewVPNBlockInterceptor refuses Click from an address in one of the vendored
// VPN ranges, and only Click: a VPN user still loads the map and follows the
// websocket, they just cannot paint. The point is not the VPN itself but the
// rate limiter behind it — a bucket keyed on an address the player can change
// at will is a bucket that does not hold.
//
// It belongs *before* NewRateLimitInterceptor in the chain. A refused address
// should not also spend a token, or the 403 would decay into a 429 on the next
// click and the web app would show the wrong dialog.
func NewVPNBlockInterceptor(blocklist ClickBlocklist, registerer prometheus.Registerer) (connect.Interceptor, error) {
	// Labelled by list, which is the only useful cut: it is how you see what
	// enabling the datacenter list would cost before enabling it.
	blocked := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "blocked_clicks",
		Help: "Clicks refused because the source address is in a blocked range",
	}, []string{"list"})

	if err := registerer.Register(blocked); err != nil {
		return nil, fmt.Errorf("failed to register counter: %w", err)
	}

	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if req.Spec().Procedure != planetv1connect.ClickServiceClickProcedure {
				return next(ctx, req)
			}

			// An address the middleware could not read is let through. Refusing
			// on a key we do not have would punish whoever the misconfiguration
			// hit, and the lists exist to refuse the known-bad, not the unknown.
			ip := ctxutil.GetSourceIP(ctx)
			if ip == "" {
				return next(ctx, req)
			}

			list, isBlocked := blocklist.Blocked(ip)
			if !isBlocked {
				return next(ctx, req)
			}

			blocked.WithLabelValues(string(list)).Inc()

			return nil, connect.NewError(connect.CodePermissionDenied, ErrVPNBlocked)
		}
	}), nil
}
