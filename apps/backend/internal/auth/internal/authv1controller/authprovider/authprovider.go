// Package authprovider names a provider and an intent on the wire, for every handler that says one.
package authprovider

import (
	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
)

var names = map[authv1.Provider]string{
	authv1.Provider_PROVIDER_GOOGLE:  signin.Google,
	authv1.Provider_PROVIDER_DISCORD: signin.Discord,
	authv1.Provider_PROVIDER_EMAIL:   signin.Email,
}

// NameOf is the provider's name, or empty for one this server has no name for.
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

// IntentOf reads only a link as a link: unset, as from every client before intents, signs in.
func IntentOf(intent authv1.SignInIntent) accounts.Intent {
	if intent == authv1.SignInIntent_SIGN_IN_INTENT_LINK {
		return accounts.IntentLink
	}
	return accounts.IntentSignIn
}
