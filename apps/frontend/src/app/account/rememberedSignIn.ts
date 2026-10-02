import {Intent, OAuthProvider} from "../../backends/account.ts"

const PROVIDER_KEY = "clickplanet-sign-in-provider"
const INTENT_KEY = "clickplanet-sign-in-intent"

const OAUTH_PROVIDERS: readonly OAuthProvider[] = ["google", "discord"]

export type RememberedSignIn = {provider: OAuthProvider, intent: Intent}

export function rememberSignIn(provider: OAuthProvider, intent: Intent) {
    try {
        sessionStorage.setItem(PROVIDER_KEY, provider)
        sessionStorage.setItem(INTENT_KEY, intent)
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
