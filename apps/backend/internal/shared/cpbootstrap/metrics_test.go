package cpbootstrap_test

import (
	"io"
	"net"
	"net/http"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

func countingModule(name, counter string, own **prometheus.Registry) cpbootstrap.Module {
	return newModule(name, func(props cpbootstrap.Props) error {
		*own = props.Metrics
		promauto.With(props.Metrics).NewCounter(prometheus.CounterOpts{Name: counter, Help: "h"}).Add(7)
		return nil
	})
}

func namesIn(t *testing.T, registry *prometheus.Registry) []string {
	t.Helper()
	families, err := registry.Gather()
	require.NoError(t, err)
	names := make([]string, 0, len(families))
	for _, family := range families {
		names = append(names, family.GetName())
	}
	return names
}

func TestMetricsServesEveryModulesCountersAndAModuleHoldsItsOwnAlone(t *testing.T) {
	public := listen(t)
	server := cpbootstrap.ServerConfig{BindAddress: public.Addr().String()}
	var planet, antibot *prometheus.Registry

	serveUntil(t, server, []net.Listener{public}, func() {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+server.BindAddress+"/metrics", nil)
		require.NoError(t, err)
		req.Close = true
		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.NoError(t, res.Body.Close())

		assert.Contains(t, string(body), "\nthrottled_clicks 7")
		assert.Contains(t, string(body), "\nbanned_callers 7")
	}, countingModule("planet", "throttled_clicks", &planet), countingModule("antibot", "banned_callers", &antibot))

	assert.Equal(t, []string{"throttled_clicks"}, namesIn(t, planet))
	assert.Equal(t, []string{"banned_callers"}, namesIn(t, antibot))
}
