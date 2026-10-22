package auth

import (
	"errors"
	"fmt"
	"net/url"
)

// The authorization server an app such as Claude connects to the admin services through. See internal/connections.
type ConnectorConfig struct {
	Enabled bool

	// The API's public origin, where an app finds the authorization server: https://api.clickplanet.lol.
	Issuer string

	// Where an app may be sent back to once the operator lets it in. Claude's callbacks unless set.
	RedirectURIs []string
}

var claudeCallbacks = []string{
	"https://claude.ai/api/mcp/auth_callback",
	"https://claude.com/api/mcp/auth_callback",
}

func (c ConnectorConfig) withDefaults() ConnectorConfig {
	if len(c.RedirectURIs) == 0 {
		c.RedirectURIs = claudeCallbacks
	}
	return c
}

// The operator lets an app in on the admin page, so there is no connector without one.
func (c ConnectorConfig) validate(admin AdminConfig) error {
	if !c.Enabled {
		return nil
	}

	var errs []error
	if admin.Origin == "" {
		errs = append(errs, errors.New("auth.connector.enabled needs auth.admin.origin: the operator lets an app in on the admin page"))
	}
	parsed, err := url.Parse(c.Issuer)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" ||
		parsed.String() != parsed.Scheme+"://"+parsed.Host {
		errs = append(errs, fmt.Errorf("auth.connector.issuer %q is not an origin: want scheme://host[:port], with no path", c.Issuer))
	}
	for _, uri := range c.RedirectURIs {
		redirect, err := url.Parse(uri)
		if err != nil || !redirect.IsAbs() || redirect.Host == "" || redirect.Fragment != "" {
			errs = append(errs, fmt.Errorf("auth.connector.redirectUris holds %q, which is not an absolute URL", uri))
		}
	}

	return errors.Join(errs...)
}
