export const CONNECT_PATH = "/connect"

const KEY = "cp-connect-request"

export function requestOf(url: URL): string | undefined {
    if (url.pathname.replace(/\/+$/, "") !== CONNECT_PATH) return undefined
    return url.searchParams.get("request") ?? ""
}

export function savePending(request: string) {
    try {
        sessionStorage.setItem(KEY, request)
    } catch {
        // A private window with storage blocked: the connection is asked for again after the sign-in.
    }
}

export function pendingRequest(): string | undefined {
    try {
        return sessionStorage.getItem(KEY) ?? undefined
    } catch {
        return undefined
    }
}

export function clearPending() {
    try {
        sessionStorage.removeItem(KEY)
    } catch {
        // Nothing was kept.
    }
}
