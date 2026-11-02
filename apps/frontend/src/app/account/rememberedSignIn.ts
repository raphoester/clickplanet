import {Intent, OAuthProvider} from "../../backends/account.ts"

const PROVIDER_KEY = "clickplanet-sign-in-provider"
const INTENT_KEY = "clickplanet-sign-in-intent"
const TRIP_KEY = "clickplanet-callback-trip"

const OAUTH_PROVIDERS: readonly OAuthProvider[] = ["google", "discord"]

export type RememberedSignIn = {provider: OAuthProvider, intent: Intent}

export type Trip = "signIn" | "discordLink"

export function rememberSignIn(provider: OAuthProvider, intent: Intent) {
    try {
        sessionStorage.setItem(PROVIDER_KEY, provider)
        sessionStorage.setItem(INTENT_KEY, intent)
        sessionStorage.removeItem(TRIP_KEY)
    } catch {
        // storage unavailable
    }
}

export function rememberedSignIn(): RememberedSignIn | undefined {
    try {
        const provider = OAUTH_PROVIDERS.find((p) => p === sessionStorage.getItem(PROVIDER_KEY))
        if (!provider) return undefined
        return {provider, intent: sessionStorage.getItem(INTENT_KEY) === "link" ? "link" : "signIn"}
    } catch {
        return undefined
    }
}

export function rememberDiscordLink() {
    try {
        sessionStorage.setItem(TRIP_KEY, "discordLink")
    } catch {
        // storage unavailable
    }
}

// The redirect URI is sign-in's: what the page left for says which trade the code is for.
export function rememberedTrip(): Trip {
    try {
        return sessionStorage.getItem(TRIP_KEY) === "discordLink" ? "discordLink" : "signIn"
    } catch {
        return "signIn"
    }
}
