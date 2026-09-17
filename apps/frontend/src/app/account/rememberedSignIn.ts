import {Intent, Provider, PROVIDERS} from "../../backends/account.ts"

const PROVIDER_KEY = "clickplanet-sign-in-provider"
const INTENT_KEY = "clickplanet-sign-in-intent"

export type RememberedSignIn = {provider: Provider, intent: Intent}

/**
 * The provider a sign-in went to and what for, for the callback page. Session
 * storage: it has to survive the trip to the provider in this tab, and nothing
 * else. It is not a secret — the flow's secrets are in the server's cookie.
 */
export function rememberSignIn(provider: Provider, intent: Intent) {
    try {
        sessionStorage.setItem(PROVIDER_KEY, provider)
        sessionStorage.setItem(INTENT_KEY, intent)
    } catch {
        // Storage off: the callback page offers no "Try again" after a spent code.
    }
}

/** A tab that left before intents were remembered went to sign in. */
export function rememberedSignIn(): RememberedSignIn | undefined {
    try {
        const provider = PROVIDERS.find((p) => p === sessionStorage.getItem(PROVIDER_KEY))
        if (!provider) return undefined
        return {provider, intent: sessionStorage.getItem(INTENT_KEY) === "link" ? "link" : "signIn"}
    } catch {
        return undefined
    }
}
