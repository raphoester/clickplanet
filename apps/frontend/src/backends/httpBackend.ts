import {Ownerships, OwnershipsGetter, TileClicker, Update, UpdatesListener} from "./backend.ts";
import {
    ClickRequest, OwnershipBatchRequest,
    Ownerships as OwnershipsProto, TileUpdate,
} from "../gen/grpc/clicks/v1/clicks_pb.ts";
import {Message} from "@bufbuild/protobuf";
import {v4 as generateUUID} from 'uuid';

type Config = {
    baseUrl: string
    timeoutMs?: number
}

const FETCH_ATTEMPTS = 5

export class ClickServiceClient {
    constructor(public config: Config) {
    }

    public async fetch(
        verb: string,
        path: string,
        body?: Message,
        signal?: AbortSignal,
    ): Promise<Uint8Array | undefined> {
        const url = this.config.baseUrl + path
        const payload = body ? JSON.stringify({data: Array.from(body.toBinary())}) : null

        let lastError: unknown
        for (let attempt = 0; attempt < FETCH_ATTEMPTS; attempt++) {
            signal?.throwIfAborted()

            let res: Response
            try {
                res = await fetch(url, {
                    method: verb,
                    headers: {'Content-Type': 'application/json'},
                    body: payload,
                    signal: timeoutSignal(this.config.timeoutMs ?? 5000, signal),
                })
            } catch (e) {
                signal?.throwIfAborted()
                lastError = e
                console.error(`${verb} ${path} failed (attempt ${attempt + 1}/${FETCH_ATTEMPTS})`, e)
                continue
            }

            if (!res.ok) {
                throw new Error(`Failed to fetch ${verb} ${path}: ${res.status} ${res.statusText} ${await res.text()}`)
            }

            const {data} = await res.json()
            return data ? decodeBase64(data) : undefined
        }

        throw new Error(`Failed to fetch ${verb} ${path} after ${FETCH_ATTEMPTS} attempts`, {cause: lastError})
    }
}

export class HTTPBackend implements TileClicker, OwnershipsGetter, UpdatesListener {
    private pendingUpdates: Update[] = []
    private readonly updateBatchCallbacks = new Map<string, (updates: Update[]) => void>()
    private readonly flushTimer: ReturnType<typeof setInterval>
    private readonly stopListening: () => void

    constructor(
        private client: ClickServiceClient,
        batchUpdateDurationMs: number,
    ) {
        this.stopListening = this.listenForUpdates((update) => {
            this.pendingUpdates.push(update)
        })

        this.flushTimer = setInterval(() => {
            if (this.pendingUpdates.length === 0) return
            const updates = this.pendingUpdates
            this.pendingUpdates = []
            this.updateBatchCallbacks.forEach(callback => callback(updates))
        }, batchUpdateDurationMs)
    }

    /** Releases the socket and the flush timer. Safe to call more than once. */
    public close() {
        clearInterval(this.flushTimer)
        this.stopListening()
        this.updateBatchCallbacks.clear()
        this.pendingUpdates = []
    }

    public async clickTile(tileId: number, countryId: string) {
        await this.client.fetch("POST", "/v2/rpc/click", new ClickRequest({
            tileId: tileId,
            countryId: countryId,
        }))
    }

    public async getCurrentOwnershipsByBatch(
        batchSize: number,
        maxIndex: number,
        callback: (ownerships: Ownerships) => void,
        signal?: AbortSignal,
    ) {
        for (let start = 1; start <= maxIndex; start += batchSize) {
            const payload = new OwnershipBatchRequest({
                startTileId: start,
                endTileId: Math.min(start + batchSize, maxIndex + 1),
            })

            const binary = await this.client.fetch("POST", "/v2/rpc/ownerships-by-batch", payload, signal)
            if (!binary) continue

            const message = OwnershipsProto.fromBinary(binary)
            callback({
                bindings: new Map<number, string>(
                    Object.entries(message.bindings).map(([k, v]) => [parseInt(k), v])),
            })
        }
    }

    public listenForUpdates(callback: (update: Update) => void): () => void {
        return openUpdatesSocket(websocketUrl(this.client.config.baseUrl), callback)
    }

    public listenForUpdatesBatch(
        callback: (updates: Update[]) => void,
    ): () => void {
        const id = generateUUID()
        this.updateBatchCallbacks.set(id, callback)
        return () => this.updateBatchCallbacks.delete(id)
    }
}

export function websocketUrl(baseUrl: string): string {
    return baseUrl.replace(/^http/, "ws") + "/v2/ws/listen"
}

/**
 * Aborts on whichever comes first: the caller giving up, or the per-request
 * timeout. `AbortSignal.timeout` alone would ignore the caller's signal, which
 * is how a teardown used to leave the remaining ownership batches in flight.
 */
function timeoutSignal(timeoutMs: number, signal?: AbortSignal): AbortSignal {
    const timeout = AbortSignal.timeout(timeoutMs)
    return signal ? AbortSignal.any([signal, timeout]) : timeout
}

function decodeBase64(base64String: string): Uint8Array {
    const binaryString = atob(base64String)
    const bytes = new Uint8Array(binaryString.length)
    for (let i = 0; i < binaryString.length; i++) {
        bytes[i] = binaryString.charCodeAt(i)
    }
    return bytes
}

const INITIAL_RECONNECT_DELAY_MS = 500
const MAX_RECONNECT_DELAY_MS = 30_000

/**
 * Holds a websocket to the updates endpoint open for as long as the caller
 * wants it, reconnecting with a capped exponential backoff.
 *
 * The socket is the only source of live tile changes, so a drop that is never
 * retried leaves the globe frozen until the user reloads. The previous version
 * connected once, rejected on the first error with nobody awaiting the promise,
 * and returned a "close" function that referenced `websocket.close` without
 * calling it — so the socket was neither retried nor released.
 *
 * Returns a function that closes the socket and cancels any pending retry.
 */
export function openUpdatesSocket(
    url: string,
    onUpdate: (update: Update) => void,
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
            const update = decodeTileUpdate(event.data)
            if (update) onUpdate(update)
        }

        /** `onerror` is always followed by `onclose`, so only one of them retries. */
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

/** A frame we cannot parse is dropped: it must not take the socket down with it. */
export function decodeTileUpdate(data: unknown): Update | undefined {
    if (!(data instanceof ArrayBuffer)) {
        console.error("Ignoring a non-binary websocket frame", data)
        return undefined
    }

    try {
        const message = TileUpdate.fromBinary(new Uint8Array(data))
        return {
            tile: message.tileId,
            previousCountry: message.previousCountryId === "" ? undefined : message.previousCountryId,
            newCountry: message.countryId,
        }
    } catch (e) {
        console.error("Ignoring a malformed tile update frame", e)
        return undefined
    }
}
