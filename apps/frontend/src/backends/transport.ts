import {Code, ConnectError} from "@connectrpc/connect";

export type Config = {
    baseUrl: string
    timeoutMs?: number
}

// connect-web applies defaultTimeoutMs to streams too; <= 0 turns it off for a call.
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
