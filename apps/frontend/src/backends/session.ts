export interface SessionProvider {
    token(): Promise<string | undefined>

    held(): string | undefined

    identity(): Promise<string | undefined>

    heldIdentity(): string | undefined

    invalidate(): void
}

export const SESSION_HEADER = "X-Session-Token"

export class SessionUnavailableError extends Error {
    constructor(options?: {cause?: unknown}) {
        super("could not start a session", options)
        this.name = "SessionUnavailableError"
    }
}

export class NoSession implements SessionProvider {
    public async token(): Promise<string | undefined> {
        return undefined
    }

    public held(): string | undefined {
        return undefined
    }

    public async identity(): Promise<string | undefined> {
        return undefined
    }

    public heldIdentity(): string | undefined {
        return undefined
    }

    public invalidate(): void {
    }
}
