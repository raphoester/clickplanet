package cpprom

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRegistry() *prometheus.Registry {
	return prometheus.NewRegistry()
}

// The process's registry and every module's, as one page.
func HandlerFor(process *prometheus.Registry, modules ...*prometheus.Registry) http.Handler {
	gatherers := make(prometheus.Gatherers, 0, 1+len(modules))
	gatherers = append(gatherers, process)
	for _, module := range modules {
		gatherers = append(gatherers, module)
	}

	return promhttp.HandlerFor(gatherers, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
		Registry:          process,
	})
}
