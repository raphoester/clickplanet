//go:build testing

package embedded_board

import (
	"log/slog"
	"sync"

	mapdata "github.com/raphoester/clickplanet.lol-backend/generated/map"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpboards"
)

var shipped = sync.OnceValues(func() (*Registry, error) {
	registry := New(cpboards.Files(), mapdata.Carries(), func(string) bool { return true }, slog.New(slog.DiscardHandler))
	return registry, registry.Load()
})

// Shipped is the ground of a board this app carries, every capital kept, loaded once for the test binary: what a test
// of the real map reads.
func Shipped(name clicks.BoardName) (*clicks.Ground, error) {
	registry, err := shipped()
	if err != nil {
		return nil, err
	}
	return registry.Ground(name)
}
