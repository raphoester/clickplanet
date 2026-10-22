package mcp_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/mcp"
)

func TestTheConnectorsURLIsAnAbsoluteURLWithAPath(t *testing.T) {
	for address, ok := range map[string]bool{
		"https://api.clickplanet.lol/mcp":   true,
		"http://localhost:8083/mcp":         true,
		"https://api.clickplanet.lol":       false,
		"https://api.clickplanet.lol/":      false,
		"/mcp":                              false,
		"https://api.clickplanet.lol/mcp?x": false,
	} {
		err := mcp.Config{Enabled: true, URL: address}.Validate()
		if ok {
			require.NoError(t, err, address)
		} else {
			require.ErrorContains(t, err, "mcp.url", address)
		}
	}

	require.NoError(t, mcp.Config{}.Validate(), "off asks for nothing")
}
