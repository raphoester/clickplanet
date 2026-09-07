import {Ownerships, OwnershipsGetter, TileClicker, Update, UpdatesListener} from "./backend.ts";
import {v4 as UUIDv4} from 'uuid';
import {Countries} from "../domain/countries.ts";

const TILE_COUNT = 257_000

export class FakeBackend implements TileClicker, OwnershipsGetter, UpdatesListener {
    private tileBindings: Map<number, string> = new Map()
    private updateListeners: Map<string, (update: Update) => void> = new Map()
    private pendingUpdates: Update[] = []
    private updateBatchCallbacks: Map<string, (update: Update[]) => void> = new Map()
    private readonly timers: ReturnType<typeof setInterval>[] = []

    constructor(batchUpdateDurationMs: number) {
        for (let i = 1; i <= TILE_COUNT; i++) {
            this.tileBindings.set(i, "fr")
        }

        this.listenForUpdates((update) => {
            this.pendingUpdates.push(update)
        })

        this.timers.push(setInterval(() => {
            if (this.pendingUpdates.length === 0) return
            const updates = this.pendingUpdates
            this.pendingUpdates = []
            this.updateBatchCallbacks.forEach(callback => callback(updates))
        }, batchUpdateDurationMs))

        Countries.forEach((country) => {
            let tileId = Math.floor(Math.random() * 10_000)
            const gap = Math.floor(Math.random() * 100)

            this.timers.push(setInterval(() => { // simulate updates
                tileId = (tileId + gap) % TILE_COUNT + 1
                this.clickTile(tileId, country.code).catch(e => console.error("failed to click", e))
            }, Math.random() * 1000))
        })
    }

    public close() {
        this.timers.forEach(clearInterval)
        this.timers.length = 0
        this.updateListeners.clear()
        this.updateBatchCallbacks.clear()
    }

    public async clickTile(tileId: number, countryId: string) {
        const prev = this.tileBindings.get(tileId)
        this.tileBindings.set(tileId, countryId)
        this.updateListeners.forEach(l => l({
            tile: tileId,
            previousCountry: prev,
            newCountry: countryId,
        }))
    }

    public listenForUpdates(
        callback: (update: Update) => void
    ): () => void {
        const identifier = UUIDv4()
        this.updateListeners.set(identifier, callback)
        return () => {
            this.updateListeners.delete(identifier)
        }
    }

    public listenForUpdatesBatch(
        callback: (updates: Update[]) => void,
    ): () => void {
        const id = UUIDv4()
        this.updateBatchCallbacks.set(id, callback)
        return () => {
            this.updateBatchCallbacks.delete(id)
        }
    }

    public async getCurrentOwnershipsByBatch(
        batchSize: number,
        maxIndex: number,
        callback: (ownerships: Ownerships) => void,
        signal?: AbortSignal,
    ) {
        for (let start = 1; start <= maxIndex; start += batchSize) {
            signal?.throwIfAborted()

            const bindings = new Map<number, string>()
            const end = Math.min(start + batchSize, maxIndex + 1)
            for (let tile = start; tile < end; tile++) {
                const owner = this.tileBindings.get(tile)
                if (owner) bindings.set(tile, owner)
            }
            callback({bindings})
        }
    }
}
