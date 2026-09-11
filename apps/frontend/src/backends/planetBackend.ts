import {
    Ownerships,
    OwnershipsGetter,
    RateLimitedError,
    TileClicker,
    Update,
    UpdatesListener,
    VPNBlockedError,
} from "./backend.ts";
import {GetMapResponse, PlanetEvent} from "../gen/grpc/planet/v1/planet_pb.ts";
import {ClickService} from "../gen/grpc/planet/v1/planet_connect.ts";
import {SessionService} from "../gen/grpc/session/v1/session_connect.ts";
import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect";
import {createConnectTransport} from "@connectrpc/connect-web";
import {v4 as generateUUID} from 'uuid';
import {Config, NO_TIMEOUT, openStream, retrying} from "./transport.ts";
import {NoSession, SESSION_HEADER, SessionProvider, SessionUnavailableError} from "./session.ts";

export type {Config}

export function newClickServiceClient(config: Config): PromiseClient<typeof ClickService> {
    return createPromiseClient(ClickService, createConnectTransport({
        baseUrl: config.baseUrl,
        useBinaryFormat: true,
        useHttpGet: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}

/**
 * Minting is a POST that must not be cached and is not on the click path's
 * critical timing, so it takes neither of the click transport's two options.
 */
export function newSessionServiceClient(config: Config): PromiseClient<typeof SessionService> {
    return createPromiseClient(SessionService, createConnectTransport({
        baseUrl: config.baseUrl,
        useBinaryFormat: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}

export class PlanetBackend implements TileClicker, OwnershipsGetter, UpdatesListener {
    private pendingUpdates: Update[] = []
    private readonly updateBatchCallbacks = new Map<string, (updates: Update[]) => void>()
    private readonly flushTimer: ReturnType<typeof setInterval>
    private readonly stopListening: () => void

    constructor(
        private client: PromiseClient<typeof ClickService>,
        batchUpdateDurationMs: number,
        private session: SessionProvider = new NoSession(),
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

    /**
     * A click the server refused for its session is retried once against a
     * freshly minted one, and the player never learns it happened: a token that
     * lapsed mid-session, or one bound to an address that changed when a phone
     * moved onto cellular, is not something to raise a dialog about. The retry
     * is not a loop — a second refusal is reported.
     */
    public async clickTile(tileId: number, countryId: string) {
        try {
            await this.click(tileId, countryId)
        } catch (e) {
            if (!(e instanceof ConnectError) || e.code !== Code.Unauthenticated) {
                throw asClickError(e)
            }

            this.session.invalidate()

            try {
                await this.click(tileId, countryId)
            } catch (retried) {
                throw asClickError(retried)
            }
        }
    }

    private async click(tileId: number, countryId: string): Promise<void> {
        const token = await this.session.token()

        const headers = new Headers()
        if (token) headers.set(SESSION_HEADER, token)

        await retrying(() => this.client.click({tileId, countryId}, {headers}), `click ${tileId}`)
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
        return openStream(
            (signal) => this.client.listenForEvents({}, {signal, timeoutMs: NO_TIMEOUT}),
            (event) => {
                const update = updateOf(event)
                if (update) callback(update)
            },
            "tile updates",
        )
    }

    public listenForUpdatesBatch(
        callback: (updates: Update[]) => void,
    ): () => void {
        const id = generateUUID()
        this.updateBatchCallbacks.set(id, callback)
        return () => this.updateBatchCallbacks.delete(id)
    }
}

export function asClickError(e: unknown): unknown {
    if (e instanceof SessionUnavailableError) return e

    if (e instanceof ConnectError) {
        if (e.code === Code.ResourceExhausted) return new RateLimitedError({cause: e})
        if (e.code === Code.PermissionDenied) return new VPNBlockedError({cause: e})
        if (e.code === Code.Unauthenticated) return new SessionUnavailableError({cause: e})
    }

    return e
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

/**
 * Anything that is not a tile update is dropped, heartbeats included. An event
 * case this build does not know reads as an unset `oneof` and lands here too,
 * which is what lets the backend add one without breaking a deployed client.
 */
export function updateOf(event: PlanetEvent): Update | undefined {
    if (event.event.case !== "tileUpdate") return undefined

    const update = event.event.value
    return {
        tile: update.tileId,
        previousCountry: update.previousCountryId === "" ? undefined : update.previousCountryId,
        newCountry: update.countryId,
    }
}
