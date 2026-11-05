export const ACCOUNT_SESSION_HEADER = "X-Account-Session"

const NO_ACCOUNT_SESSION = "none"

export const ACCOUNT_STORAGE_KEY = "clickplanet-account"

export type AccountCarrier = {
    read(): string | undefined
    write(session: string | undefined): void
}

// A page framed on another site keeps no cookie of the API's, Safari not even a partitioned one: the page carries it.
export function carriedFetch(carrier: AccountCarrier, fetch: typeof globalThis.fetch = (input, init) => globalThis.fetch(input, init)): typeof globalThis.fetch {
    return async (input, init) => {
        const headers = new Headers(init?.headers)
        headers.set(ACCOUNT_SESSION_HEADER, carrier.read() ?? NO_ACCOUNT_SESSION)
        const res = await fetch(input, {...init, headers, credentials: "omit"})
        const answered = res.headers.get(ACCOUNT_SESSION_HEADER)
        if (answered !== null) carrier.write(answered === NO_ACCOUNT_SESSION ? undefined : answered)
        return res
    }
}

export function localAccountCarrier(): AccountCarrier {
    return {
        read(): string | undefined {
            try {
                return window.localStorage.getItem(ACCOUNT_STORAGE_KEY) ?? undefined
            } catch {
                return undefined
            }
        },
        write(session: string | undefined): void {
            try {
                if (session === undefined) window.localStorage.removeItem(ACCOUNT_STORAGE_KEY)
                else window.localStorage.setItem(ACCOUNT_STORAGE_KEY, session)
            } catch {
                // storage unavailable
            }
        },
    }
}
