/** Where every provider sends the browser back. Registered with each provider exactly. */
export const CALLBACK_PATH = "/auth/callback"

export type SignInCallback =
    /** The provider sent a code to trade. */
    | {kind: "code", code: string, state: string}
    /** The player said no on the provider's page, or the provider refused. */
    | {kind: "declined"}
    /** Opened by hand, or with a part missing. */
    | {kind: "invalid"}

/**
 * What the provider sent to the callback page, or undefined on any other page.
 * An `error` wins over a code: OAuth sends `error=access_denied` when the
 * player cancels, and nothing that came with it is worth trading.
 */
export function callbackOf(url: URL): SignInCallback | undefined {
    if (url.pathname.replace(/\/+$/, "") !== CALLBACK_PATH) return undefined

    const params = url.searchParams
    if (params.has("error")) return {kind: "declined"}

    const code = params.get("code")
    const state = params.get("state")
    if (!code || !state) return {kind: "invalid"}

    return {kind: "code", code, state}
}
