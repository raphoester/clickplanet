package inmemory_tile_storage

import (
	"fmt"
	"math"
)

const maxCodes = math.MaxUint16 + 1

const unownedCode = uint16(0)

// Country codes interned to two bytes, so a tile's owner is an index and not a string.
type codebook struct {
	names []string
	ids   map[string]uint16
}

func newCodebook() *codebook {
	return &codebook{names: []string{""}, ids: map[string]uint16{"": unownedCode}}
}

func (c *codebook) intern(name string) (uint16, error) {
	if id, ok := c.ids[name]; ok {
		return id, nil
	}

	if len(c.names) >= maxCodes {
		return 0, fmt.Errorf("country code table is full (%d entries)", maxCodes)
	}

	id := uint16(len(c.names)) //nolint:gosec // under maxCodes.
	c.names = append(c.names, name)
	c.ids[name] = id

	return id, nil
}

func (c *codebook) idOf(name string) (uint16, bool) {
	id, ok := c.ids[name]
	return id, ok
}

func (c *codebook) nameOf(id uint16) string {
	return c.names[id]
}

func (c *codebook) all() []string {
	names := make([]string, len(c.names))
	copy(names, c.names)
	return names
}

func (c *codebook) countries() int {
	return len(c.names) - 1
}
