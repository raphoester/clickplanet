// The backend's authorize endpoint sends the operator's browser here, with the request an app made, sealed.
export const CONNECT_PATH = "/connect"

const KEY = "cp-connect-request"

export function requestOf(url: URL): string | undefined {
    if (url.pathname.replace(/\/+$/, "") !== CONNECT_PATH) return undefined
    return url.searchParams.get("request") ?? ""
}

// Kept for this tab while the operator signs in: Google comes back to /auth/callback, not here. The request lasts
// ten minutes on the server, so a stale one is refused there. Storage can be off, and then the operator connects again.
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
