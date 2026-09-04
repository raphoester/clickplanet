package app

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/httpserver"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/prom"
)

func (a *App) configureHTTPFormatsIfNeeded() {
	if a.reader != nil && a.answerer != nil {
		return
	}

	format := httpserver.FormatFromString(a.config.HTTPServer.Format)
	answerer, reader := format.Build(a.logger)

	a.answerer = answerer
	a.reader = reader
}

func (a *App) configurePromRegistryIfNeeded() {
	if a.promRegistry != nil {
		return
	}

	a.promRegistry = prom.NewRegistry()
}
