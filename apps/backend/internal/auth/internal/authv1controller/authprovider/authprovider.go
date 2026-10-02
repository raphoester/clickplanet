package authprovider

import (
	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
)

var names = map[authv1.Provider]string{
	authv1.Provider_PROVIDER_GOOGLE:  signin.Google,
	authv1.Provider_PROVIDER_DISCORD: signin.Discord,
}

func NameOf(provider authv1.Provider) string {
	return names[provider]
}

func ProtoOf(name string) authv1.Provider {
	for provider, known := range names {
		if known == name {
			return provider
		}
	}
	return authv1.Provider_PROVIDER_UNSPECIFIED
}
