// Package authprovider names a provider on the wire, for every handler that says one.
package authprovider

import (
	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
)

var names = map[authv1.Provider]string{
	authv1.Provider_PROVIDER_GOOGLE:  signin.Google,
	authv1.Provider_PROVIDER_DISCORD: signin.Discord,
}

// Decode is the provider's name, or empty for one this server has no name for.
func Decode(provider authv1.Provider) string {
	return names[provider]
}

func Encode(name string) authv1.Provider {
	for provider, known := range names {
		if known == name {
			return provider
		}
	}
	return authv1.Provider_PROVIDER_UNSPECIFIED
}
