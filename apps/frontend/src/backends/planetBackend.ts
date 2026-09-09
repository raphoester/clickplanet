import {
    Ownerships,
    OwnershipsGetter,
    RateLimitedError,
    TileClicker,
    Update,
    UpdatesListener,
    VPNBlockedError,
} from "./backend.ts";
import {GetMapResponse, TileUpdate} from "../gen/grpc/planet/v1/planet_pb.ts";
import {ClickService} from "../gen/grpc/planet/v1/planet_connect.ts";
import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect";
import {createConnectTransport} from "@connectrpc/connect-web";
import {v4 as generateUUID} from 'uuid';

export type Config = {
    baseUrl: string
    timeoutMs?: number
}

const ATTEMPTS = 5

export function websocketUrl(baseUrl: string): string {
    return baseUrl.replace(/^http/, "ws") + "/ws/listen"
}

export function newClickServiceClient(config: Config): PromiseClient<typeof ClickService> {
    return createPromiseClient(ClickService, createConnectTransport({
        baseUrl: config.baseUrl,
        useBinaryFormat: true,
        useHttpGet: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}

export class PlanetBackend implements TileClicker, OwnershipsGetter, UpdatesListener {
    private pendingUpdates: Update[] = []
    private readonly updateBatchCallbacks = new Map<string, (updates: Update[]) => void>()
    private readonly flushTimer: ReturnType<typeof setInterval>
    private readonly stopListening: () => void

    constructor(
        private config: Config,
        private client: PromiseClient<typeof ClickService>,
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

    public close() {
        clearInterval(this.flushTimer)
        this.stopListening()
        this.updateBatchCallbacks.clear()
        this.pendingUpdates = []
    }

    public async clickTile(tileId: number, countryId: string) {
        try {
            await retrying(() => this.client.click({tileId, countryId}), `click ${tileId}`)
        } catch (e) {
            if (e instanceof ConnectError && e.code === Code.ResourceExhausted) {
                throw new RateLimitedError({cause: e})
            }
            if (e instanceof ConnectError && e.code === Code.PermissionDenied) {
                throw new VPNBlockedError({cause: e})
            }
            throw e
        }
    }

    public async getCurrentOwnershipsByBatch(
        batchSize: number,
        maxIndex: number,
        callback: (ownerships: Ownerships) => void,
        signal?: AbortSignal,
    ) {
        for (let start = 1; start <= maxIndex; start += batchSize) {
            const endTileId = Math.min(start + batchSize, maxIndex)

            const res = await retrying(
                () => this.client.getMap({startTileId: start, endTileId}, {signal}),
                `getMap ${start}..${endTileId}`,
                signal,
            )

            callback({bindings: bindingsOf(res)})
        }
    }

    public listenForUpdates(callback: (update: Update) => void): () => void {
        return openUpdatesSocket(websocketUrl(this.config.baseUrl), callback)
    }

    public listenForUpdatesBatch(
        callback: (updates: Update[]) => void,
    ): () => void {
        const id = generateUUID()
        this.updateBatchCallbacks.set(id, callback)
        return () => this.updateBatchCallbacks.delete(id)
    }
}

export function bindingsOf(res: GetMapResponse): Map<number, string> {
    const tiles = new DataView(res.tiles.buffer, res.tiles.byteOffset, res.tiles.byteLength)

    const bindings = new Map<number, string>()
    for (let offset = 0; offset + 1 < res.tiles.byteLength; offset += 2) {
        const code = tiles.getUint16(offset, true)
        if (code === 0) continue
        bindings.set(res.startTileId + offset / 2, res.codes[code])
    }

    return bindings
}

async function retrying<T>(
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
