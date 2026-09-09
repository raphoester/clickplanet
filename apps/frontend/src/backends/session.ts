/**
 * The click contract's second half: the server mints a token once, and every
 * click carries it. Nothing here is an account — a session says a caller passed
 * attestation from this address, and it expires.
 */
export interface SessionProvider {
    /**
     * The token to put on the next click, or undefined when this build runs
     * against a server that does not use sessions.
     */
    token(): Promise<string | undefined>

    /** Called when the server refused the token we last supplied. */
    invalidate(): void
}

export const SESSION_HEADER = "X-Session-Token"

/**
 * A click could not be sent because no session could be obtained — Turnstile
 * did not answer, or the server refused to mint. Distinct from a refused click:
 * nothing was attempted, and retrying later may well work.
 */
export class SessionUnavailableError extends Error {
    constructor(options?: {cause?: unknown}) {
        super("could not start a session", options)
        this.name = "SessionUnavailableError"
    }
}

/** For a backend that does not use sessions, and for the fake one. */
export class NoSession implements SessionProvider {
    public async token(): Promise<string | undefined> {
        return undefined
    }

    public invalidate(): void {
    }
}
