import {Code, ConnectError, Interceptor} from "@connectrpc/connect"

export type Connection = "up" | "down"

export interface ConnectionSource {
    watchConnection(callback: (connection: Connection) => void): () => void
}

export const DOWN_AFTER_MISSES = 2

export class ConnectionHealth implements ConnectionSource {
    private misses = 0
    private connection: Connection = "up"
    private readonly callbacks = new Set<(connection: Connection) => void>()

    public reached(): void {
        this.misses = 0
        this.become("up")
    }

    public missed(): void {
        this.misses++
        if (this.misses >= DOWN_AFTER_MISSES) this.become("down")
    }

    public wentOffline(): void {
        this.misses = DOWN_AFTER_MISSES
        this.become("down")
    }

    public watchConnection(callback: (connection: Connection) => void): () => void {
        this.callbacks.add(callback)
        callback(this.connection)
        return () => {
            this.callbacks.delete(callback)
        }
    }

    private become(connection: Connection): void {
        if (connection === this.connection) return
        this.connection = connection
        this.callbacks.forEach((callback) => callback(connection))
    }
}

export function followBrowser(health: ConnectionHealth, browser: EventTarget): () => void {
    const offline = () => health.wentOffline()
    browser.addEventListener("offline", offline)
    return () => browser.removeEventListener("offline", offline)
}

export function connectionInterceptor(health: ConnectionHealth): Interceptor {
    return (next) => async (req) => {
        let res
        try {
            res = await next(req)
        } catch (e) {
            observe(health, req.signal.aborted ? req.signal.reason : e)
            throw e
        }

        health.reached()
        if (!res.stream) return res

        return {...res, message: followed(res.message, health, req.signal)}
    }
}

async function* followed<T>(messages: AsyncIterable<T>, health: ConnectionHealth, signal: AbortSignal): AsyncIterable<T> {
    try {
        for await (const message of messages) {
            health.reached()
            yield message
        }
    } catch (e) {
        observe(health, signal.aborted ? signal.reason : e)
        throw e
    }
}

function observe(health: ConnectionHealth, failure: unknown): void {
    const outcome = outcomeOf(failure)
    if (outcome === "reached") health.reached()
    if (outcome === "missed") health.missed()
}

// A ConnectError the server sent is an answer; one about the road to it is a miss.
function outcomeOf(failure: unknown): "reached" | "missed" | undefined {
    if (failure instanceof ConnectError) {
        switch (failure.code) {
            case Code.Canceled:
                return undefined
            case Code.Unavailable:
            case Code.DeadlineExceeded:
                return "missed"
            case Code.Unknown:
                return failure.cause instanceof TypeError ? "missed" : "reached"
            default:
                return "reached"
        }
    }

    if (failure instanceof Error && failure.name === "AbortError") return undefined

    return "missed"
}
