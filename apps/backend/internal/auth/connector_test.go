package auth_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth"
)

func connected() auth.Config {
	config := validConfig()
	config.Admin = adminPage("https://backoffice.clickplanet.lol")
	config.Connector = auth.ConnectorConfig{Enabled: true, Issuer: "https://api.clickplanet.lol"}
	return config
}

func TestAConnectorOnTheAPIsOriginWithAnAdminPageIsValid(t *testing.T) {
	require.NoError(t, connected().Validate(), "Claude's callbacks unless set")
}

func TestAConnectorNeedsAnAdminPageToBeLetInOn(t *testing.T) {
	config := connected()
	config.Admin = auth.AdminConfig{}

	require.ErrorContains(t, config.Validate(), "auth.admin.origin")
}

func TestTheIssuerIsAnOrigin(t *testing.T) {
	for issuer, ok := range map[string]bool{
		"https://api.clickplanet.lol":  true,
		"http://localhost:8083":        true,
		"":                             false,
		"https://api.clickplanet.lol/": false,
		"api.clickplanet.lol":          false,
	} {
		config := connected()
		config.Connector.Issuer = issuer
		if ok {
			require.NoError(t, config.Validate(), issuer)
		} else {
			require.ErrorContains(t, config.Validate(), "auth.connector.issuer", issuer)
		}
	}
}

func TestARedirectIsAnAbsoluteURL(t *testing.T) {
	config := connected()
	config.Connector.RedirectURIs = []string{"/api/mcp/auth_callback"}

	require.ErrorContains(t, config.Validate(), "auth.connector.redirectUris")
}

func TestAConnectorNeedsAuthOn(t *testing.T) {
	off := auth.Config{Connector: auth.ConnectorConfig{Enabled: true, Issuer: "https://api.clickplanet.lol"}}

	require.ErrorContains(t, off.Validate(), "auth.connector.enabled")
}
