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
import {Config, openSocket, retrying, websocketUrl as socketUrl} from "./transport.ts";

export type {Config}

const TILE_UPDATE_ROUTE = "/ws/listen"

export function websocketUrl(baseUrl: string): string {
    return socketUrl(baseUrl, TILE_UPDATE_ROUTE)
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

export function openUpdatesSocket(
    url: string,
    onUpdate: (update: Update) => void,
): () => void {
    return openSocket(url, (data) => {
        const update = decodeTileUpdate(data)
        if (update) onUpdate(update)
    })
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
