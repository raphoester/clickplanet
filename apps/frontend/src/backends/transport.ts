import {Code, ConnectError} from "@connectrpc/connect";

export type Config = {
    baseUrl: string
    timeoutMs?: number
}

export function websocketUrl(baseUrl: string, route: string): string {
    return baseUrl.replace(/^http/, "ws") + route
}

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

export function openSocket(
    url: string,
    onFrame: (data: unknown) => void,
): () => void {
    let socket: WebSocket | undefined
    let retryTimer: ReturnType<typeof setTimeout> | undefined
    let retryDelayMs = INITIAL_RECONNECT_DELAY_MS
    let stopped = false

    const scheduleReconnect = () => {
        if (stopped || retryTimer !== undefined) return
        retryTimer = setTimeout(() => {
            retryTimer = undefined
            connect()
        }, retryDelayMs)
        retryDelayMs = Math.min(retryDelayMs * 2, MAX_RECONNECT_DELAY_MS)
    }

    const connect = () => {
        if (stopped) return

        const ws = new WebSocket(url)
        socket = ws
        ws.binaryType = "arraybuffer"

        ws.onopen = () => {
            retryDelayMs = INITIAL_RECONNECT_DELAY_MS
        }

        ws.onmessage = (event) => {
            onFrame(event.data)
        }

        ws.onclose = () => {
            if (socket === ws) socket = undefined
            scheduleReconnect()
        }
    }

    connect()

    return () => {
        stopped = true
        if (retryTimer !== undefined) clearTimeout(retryTimer)
        const ws = socket
        socket = undefined
        if (ws) {
            ws.onclose = null
            ws.close()
        }
    }
}
