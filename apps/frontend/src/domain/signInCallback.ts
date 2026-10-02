export const CALLBACK_PATH = "/auth/callback"

export const GAME_PATH = "/play"

export type SignInCallback =
    | {kind: "code", code: string, state: string}
    | {kind: "declined"}
    | {kind: "invalid"}

export function callbackOf(url: URL): SignInCallback | undefined {
    if (url.pathname.replace(/\/+$/, "") !== CALLBACK_PATH) return undefined

    const params = url.searchParams
    if (params.has("error")) return {kind: "declined"}

    const code = params.get("code")
    const state = params.get("state")
    if (!code || !state) return {kind: "invalid"}

    return {kind: "code", code, state}
}
