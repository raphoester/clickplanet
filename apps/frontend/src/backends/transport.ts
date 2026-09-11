import {Code, ConnectError} from "@connectrpc/connect";

export type Config = {
    baseUrl: string
    timeoutMs?: number
}

/**
 * Passed as a call's `timeoutMs` to opt out of the transport's `defaultTimeoutMs`,
 * which connect-web otherwise applies to a stream exactly as to a unary call —
 * ending a healthy live feed after five seconds. Anything <= 0 means no timeout.
 */
export const NO_TIMEOUT = 0

const ATTEMPTS = 5

export async function retrying<T>(
    attempt: () => Promise<T>,
    what: string,
    signal?: AbortSignal,
): Promise<T> {
    let lastError: unknown

    for (let i = 0; i < ATTEMPTS; i++) {
        signal?.throwIfAborted()

        try {
            return await attempt()
        } catch (e) {
            signal?.throwIfAborted()
            if (!unreachable(e)) throw e

            lastError = e
            console.error(`${what} failed (attempt ${i + 1}/${ATTEMPTS})`, e)
        }
    }

    throw new Error(`${what} failed after ${ATTEMPTS} attempts`, {cause: lastError})
}

function unreachable(e: unknown): boolean {
    return !(e instanceof ConnectError) || e.code === Code.Unavailable
}

const INITIAL_RECONNECT_DELAY_MS = 500
const MAX_RECONNECT_DELAY_MS = 30_000

/**
 * Follows a server-streaming RPC for as long as the caller wants it, reopening
 * it with a capped exponential backoff.
 *
 * A stream is a one-shot async iterable: it ends on a dropped connection, a
 * restarted server or a proxy timeout, and nothing reopens it. That reconnect
 * loop is the whole reason this exists — Connect does not carry one, and every
 * caller would otherwise write it.
 *
 * The delay resets on a received message rather than on connect, because a
 * connection is only known to work once something has come down it.
 */
export function openStream<T>(
    open: (signal: AbortSignal) => AsyncIterable<T>,
    onMessage: (message: T) => void,
    what: string,
): () => void {
    let controller: AbortController | undefined
    let retryTimer: ReturnType<typeof setTimeout> | undefined
    let retryDelayMs = INITIAL_RECONNECT_DELAY_MS
    let stopped = false

    const scheduleReconnect = () => {
        if (stopped || retryTimer !== undefined) return
        retryTimer = setTimeout(() => {
            retryTimer = undefined
            void connect()
        }, retryDelayMs)
        retryDelayMs = Math.min(retryDelayMs * 2, MAX_RECONNECT_DELAY_MS)
    }

    const connect = async () => {
        if (stopped) return

        const attempt = new AbortController()
        controller = attempt

        try {
            for await (const message of open(attempt.signal)) {
                retryDelayMs = INITIAL_RECONNECT_DELAY_MS
                onMessage(message)
            }
        } catch (e) {
            if (stopped || attempt.signal.aborted) return
            console.error(`${what} stream failed`, e)
        }

        if (controller === attempt) controller = undefined
        scheduleReconnect()
    }

    void connect()

    return () => {
        stopped = true
        if (retryTimer !== undefined) clearTimeout(retryTimer)
        controller?.abort()
        controller = undefined
    }
}
