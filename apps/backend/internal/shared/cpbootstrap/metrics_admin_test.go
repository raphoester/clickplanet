package cpbootstrap_test

import (
	"net"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	metricsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/metrics/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/metrics/v1/metricsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

func countingModule() cpbootstrap.Module {
	return newModule("planet", func(props cpbootstrap.Props) error {
		promauto.With(props.Metrics).NewCounter(prometheus.CounterOpts{Name: "throttled_clicks", Help: "h"}).Add(7)
		return nil
	})
}

func counterOn(t *testing.T, address, name string) (float64, error) {
	t.Helper()
	res, err := metricsv1connect.NewAdminServiceClient(http.DefaultClient, "http://"+address).
		GetMetrics(t.Context(), connect.NewRequest(&metricsv1.GetMetricsRequest{}))
	if err != nil {
		return 0, err //nolint:wrapcheck // the test reads the connect code.
	}
	for _, metric := range res.Msg.GetMetrics() {
		if metric.GetName() == name {
			return metric.GetCounter(), nil
		}
	}
	return 0, nil
}

func TestTheConsoleAnswersWhatTheModulesCountAndThePublicRouterDoesNot(t *testing.T) {
	public, console := listen(t), listen(t)
	server := cpbootstrap.ServerConfig{BindAddress: public.Addr().String(), ConsoleBindAddress: console.Addr().String()}

	serveUntil(t, server, []net.Listener{public, console}, func() {
		throttled, err := counterOn(t, server.ConsoleBindAddress, "throttled_clicks")
		require.NoError(t, err)
		assert.InDelta(t, 7, throttled, 0)

		_, err = counterOn(t, server.BindAddress, "throttled_clicks")
		assert.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
	}, countingModule())
}

func TestTheAdminListenerAnswersTheMetricsBehindTheGate(t *testing.T) {
	public, admin := listen(t), listen(t)
	server := cpbootstrap.ServerConfig{BindAddress: public.Addr().String(), AdminBindAddress: admin.Addr().String()}

	serveBehind(t, server, headerGate(), []net.Listener{public, admin}, func() {
		_, err := counterOn(t, server.AdminBindAddress, "throttled_clicks")
		assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	}, countingModule())
}
