//go:build testing

package cpbootstrap

import (
	"context"
	"fmt"
	"net"
)

func RunOn(ctx context.Context, options Options, listeners ...net.Listener) error {
	defer func() {
		for _, listener := range listeners {
			_ = listener.Close()
		}
	}()

	held := make(heldListeners, len(listeners))
	for _, listener := range listeners {
		held[listener.Addr().String()] = listener
	}

	return run(ctx, options, held.listen)
}

type heldListeners map[string]net.Listener

func (h heldListeners) listen(_ context.Context, _, address string) (net.Listener, error) {
	listener, ok := h[address]
	if !ok {
		return nil, fmt.Errorf("no listener held for %q", address)
	}

	return listener, nil
}
