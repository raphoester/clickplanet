import {Code, ConnectError, Interceptor} from "@connectrpc/connect";

export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "https://api.clickplanet.lol"

export type Config = {
    baseUrl: string
    timeoutMs?: number
    interceptors?: Interceptor[]
}

// connect-web applies defaultTimeoutMs to streams too; <= 0 turns it off for a call.
export const NO_TIMEOUT = 0

type Retries = {
    attempts: number
    delayBefore: (attempt: number) => number
    worthRetrying: (e: unknown) => boolean
}

const READS: Retries = {
    attempts: 8,
    delayBefore: (attempt) => attempt < 2 ? 0 : Math.min(1000 * 2 ** (attempt - 2), 8_000),
    worthRetrying: (e) => unreachable(e) || timedOut(e),
}

const ACTS: Retries = {
    attempts: 5,
    delayBefore: () => 0,
    worthRetrying: unreachable,
}

export function retrying<T>(attempt: () => Promise<T>, what: string, signal?: AbortSignal): Promise<T> {
    return retried(attempt, what, READS, signal)
}

export function retryingAtOnce<T>(attempt: () => Promise<T>, what: string): Promise<T> {
    return retried(attempt, what, ACTS)
}

async function retried<T>(
    attempt: () => Promise<T>,
    what: string,
    {attempts, delayBefore, worthRetrying}: Retries,
    signal?: AbortSignal,
): Promise<T> {
    let lastError: unknown

    for (let i = 0; i < attempts; i++) {
        signal?.throwIfAborted()
        const delay = delayBefore(i)
        if (delay > 0) await pause(delay, signal)

        try {
            return await attempt()
        } catch (e) {
            signal?.throwIfAborted()
            if (!worthRetrying(e)) throw e

            lastError = e
            console.error(`${what} failed (attempt ${i + 1}/${attempts})`, e)
        }
    }

    throw new Error(`${what} failed after ${attempts} attempts`, {cause: lastError})
}

// connect-web turns a failed fetch into Unknown, with the fetch's TypeError as its cause.
function unreachable(e: unknown): boolean {
    if (!(e instanceof ConnectError)) return true
    return e.code === Code.Unavailable || (e.code === Code.Unknown && e.cause instanceof TypeError)
}

function timedOut(e: unknown): boolean {
    return e instanceof ConnectError && e.code === Code.DeadlineExceeded
}

function pause(ms: number, signal?: AbortSignal): Promise<void> {
    return new Promise((resolve, reject) => {
        const stop = () => {
            clearTimeout(timer)
            reject(signal?.reason)
        }
        const timer = setTimeout(() => {
            signal?.removeEventListener("abort", stop)
            resolve()
        }, ms)
        signal?.addEventListener("abort", stop, {once: true})
    })
}

const INITIAL_RECONNECT_DELAY_MS = 500
const MAX_RECONNECT_DELAY_MS = 30_000

// Two of the server's 30s heartbeats missed.
export const SILENCE_LIMIT_MS = 60_000

export type Wakeups = (wake: () => void) => () => void

export const pageWakeups: Wakeups = (wake) => {
    if (typeof document === "undefined") return () => {}

    const onVisibility = () => {
        if (document.visibilityState === "visible") wake()
    }

    document.addEventListener("visibilitychange", onVisibility)
    document.addEventListener("resume", wake)
    window.addEventListener("pageshow", wake)
    window.addEventListener("online", wake)

    return () => {
        document.removeEventListener("visibilitychange", onVisibility)
        document.removeEventListener("resume", wake)
        window.removeEventListener("pageshow", wake)
        window.removeEventListener("online", wake)
    }
}

export type StreamOptions = {
    onResumed?: () => void
    wakeups?: Wakeups
}

export function openStream<T>(
    open: (signal: AbortSignal) => AsyncIterable<T>,
    onMessage: (message: T) => void,
    what: string,
    {onResumed, wakeups = pageWakeups}: StreamOptions = {},
): () => void {
    let controller: AbortController | undefined
    let retryTimer: ReturnType<typeof setTimeout> | undefined
    let silenceTimer: ReturnType<typeof setTimeout> | undefined
    let retryDelayMs = INITIAL_RECONNECT_DELAY_MS
    let heardAt = Date.now()
    let wasLive = false
    let stopped = false

    const scheduleReconnect = () => {
        if (stopped || retryTimer !== undefined) return
        retryTimer = setTimeout(() => {
            retryTimer = undefined
            void connect()
        }, retryDelayMs)
        retryDelayMs = Math.min(retryDelayMs * 2, MAX_RECONNECT_DELAY_MS)
    }

    const silentFor = () => Date.now() - heardAt

    const watchSilence = () => {
        if (silenceTimer !== undefined) clearTimeout(silenceTimer)
        silenceTimer = setTimeout(() => {
            silenceTimer = undefined
            if (controller === undefined) return
            if (silentFor() >= SILENCE_LIMIT_MS) {
                reopen()
            } else {
                watchSilence()
            }
        }, SILENCE_LIMIT_MS - silentFor())
    }

    const reopen = () => {
        if (stopped) return

        if (retryTimer !== undefined) clearTimeout(retryTimer)
        retryTimer = undefined
        retryDelayMs = INITIAL_RECONNECT_DELAY_MS

        const stale = controller
        controller = undefined
        stale?.abort()

        void connect()
    }

    const connect = async () => {
        if (stopped) return

        const attempt = new AbortController()
        controller = attempt
        heardAt = Date.now()
        watchSilence()

        let first = true
        try {
            for await (const message of open(attempt.signal)) {
                heardAt = Date.now()
                retryDelayMs = INITIAL_RECONNECT_DELAY_MS
                if (first) {
                    first = false
                    if (wasLive) onResumed?.()
                    wasLive = true
                }
                onMessage(message)
            }
        } catch (e) {
            if (stopped || attempt.signal.aborted) return
            console.error(`${what} stream failed`, e)
        }

        if (controller !== attempt) return
        controller = undefined
        scheduleReconnect()
    }

    const stopWaking = wakeups(() => {
        if (stopped) return
        if (retryTimer !== undefined || silentFor() >= SILENCE_LIMIT_MS) reopen()
    })

    void connect()

    return () => {
        stopped = true
        stopWaking()
        if (retryTimer !== undefined) clearTimeout(retryTimer)
        if (silenceTimer !== undefined) clearTimeout(silenceTimer)
        controller?.abort()
        controller = undefined
    }
}
