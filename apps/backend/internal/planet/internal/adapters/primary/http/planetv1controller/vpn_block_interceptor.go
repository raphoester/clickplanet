package planetv1controller

import (
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/connectutil"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/ipblock"
)

var ErrVPNBlocked = errors.New("clicks from VPN and proxy addresses are refused; turn yours off to play")

type ClickBlocklist = connectutil.Blocklist

func NewVPNBlockInterceptor(blocklist ClickBlocklist, registerer prometheus.Registerer) (connect.Interceptor, error) {
	blocked := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "blocked_clicks",
		Help: "Clicks refused because the source address is in a blocked range",
	}, []string{"list"})

	if err := registerer.Register(blocked); err != nil {
		return nil, fmt.Errorf("failed to register counter: %w", err)
	}

	return connectutil.NewIPBlockInterceptor(
		blocklist,
		ErrVPNBlocked,
		func(list ipblock.List) { blocked.WithLabelValues(string(list)).Inc() },
		planetv1connect.ClickServiceClickProcedure,
	), nil
}
