package admin_server

import (
	"fmt"
	"net"
)

type Config struct {
	Enabled bool

	// Loopback only: the server rewrites the map and has no authentication.
	BindAddress string
}

const defaultBindAddress = "127.0.0.1:8081"

func (c Config) Address() string {
	if c.BindAddress == "" {
		return defaultBindAddress
	}

	return c.BindAddress
}

func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	host, _, err := net.SplitHostPort(c.Address())
	if err != nil {
		return fmt.Errorf("admin.bindAddress %q is not host:port: %w", c.Address(), err)
	}

	if host == "localhost" {
		return nil
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return nil
	}

	return fmt.Errorf(
		"admin.bindAddress %q is not a loopback address: the admin server rewrites the map with no authentication",
		c.Address(),
	)
}
