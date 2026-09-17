import {Provider, PROVIDERS} from "../../backends/account.ts"

const KEY = "clickplanet-sign-in-provider"

/**
 * The provider a sign-in went to, for the callback page's "Try again". Session
 * storage: it has to survive the trip to the provider in this tab, and nothing
 * else. It is not a secret — the flow's secrets are in the server's cookie.
 */
export function rememberProvider(provider: Provider) {
    try {
        sessionStorage.setItem(KEY, provider)
    } catch {
        // Storage off: the callback page offers no "Try again" after a spent code.
    }
}

export function rememberedProvider(): Provider | undefined {
    try {
        const value = sessionStorage.getItem(KEY)
        return PROVIDERS.find((p) => p === value)
    } catch {
        return undefined
    }
}
