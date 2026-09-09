package app

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/prom"
)

func (a *App) configurePromRegistryIfNeeded() {
	if a.promRegistry != nil {
		return
	}

	a.promRegistry = prom.NewRegistry()
}
