// The backend's auth.admin.redirectUrl: Google sends the browser back here.
export const CALLBACK_PATH = "/auth/callback"

export type Callback =
    | {kind: "code", code: string, state: string}
    | {kind: "refused"}
    | {kind: "broken"}

export function callbackOf(url: URL): Callback | undefined {
    if (url.pathname.replace(/\/+$/, "") !== CALLBACK_PATH) return undefined

    if (url.searchParams.has("error")) return {kind: "refused"}

    const code = url.searchParams.get("code") ?? ""
    const state = url.searchParams.get("state") ?? ""
    if (code === "" || state === "") return {kind: "broken"}

    return {kind: "code", code, state}
}
