import {MapReel} from "../domain/mapHistory.ts"
import {Counts, MapVersion, Ownerships, OwnershipsGetter, TileRange, Update, UpdatesListener} from "./backend.ts"
import {FakeTally} from "./fakeTally.ts"
import {ReplayLandmasses} from "./replayBackend.ts"

// A history counts its seeks from its opening, as one start of the server.
const EPOCH = 1

// Shows a map history on the globe as the live stream would: seek hands over the tiles that differ from the frame shown.
export class HistoryBackend implements OwnershipsGetter, UpdatesListener {
    private shown: Uint16Array
    private seeks = 0
    private readonly tally: FakeTally
    private readonly batchListeners = new Set<(updates: Update[]) => void>()
    private readonly holdingsListeners = new Set<(change: Counts) => void>()

    constructor(private readonly reel: MapReel, landmasses: ReplayLandmasses) {
        this.shown = reel.ownersAt(0)
        this.tally = new FakeTally(landmasses.assignment, landmasses.count)
        this.shown.forEach((owner, tile) => {
            if (owner !== 0) this.tally.moved(tile, undefined, reel.codes[owner])
        })
        this.tally.changes(this.version())
    }

    seek(frame: number) {
        const target = this.reel.ownersAt(frame)
        this.seeks++
        const version = this.version()
        const updates: Update[] = []
        for (let tile = 1; tile < target.length; tile++) {
            if (target[tile] === this.shown[tile]) continue
            const previous = this.codeOf(this.shown[tile])
            const next = this.codeOf(target[tile])
            this.tally.moved(tile, previous, next)
            updates.push({tile, previousCountry: previous, newCountry: next, clicked: false, shields: 0, version})
        }
        this.shown = target
        if (updates.length > 0) for (const listener of this.batchListeners) listener(updates)
        const change = this.tally.changes(version)
        if (change) for (const listener of this.holdingsListeners) listener(change)
    }

    async getOwnerships(ranges: readonly TileRange[], callback: (ownerships: Ownerships) => void): Promise<void> {
        for (const range of ranges) {
            const bindings = new Map<number, string>()
            for (let tile = range.start; tile <= range.end; tile++) {
                const owner = this.codeOf(this.shown[tile] ?? 0)
                if (owner !== undefined) bindings.set(tile, owner)
            }
            callback({range, version: this.version(), bindings, shields: new Map()})
        }
    }

    async getHoldings(): Promise<Counts> {
        return this.tally.read(this.version())
    }

    async getFortresses(): Promise<Map<number, string>> {
        return new Map()
    }

    listenForUpdates(): () => void {
        return () => {}
    }

    listenForUpdatesBatch(callback: (updates: Update[]) => void): () => void {
        this.batchListeners.add(callback)
        return () => this.batchListeners.delete(callback)
    }

    listenForOpenings(callback: (version: MapVersion) => void): () => void {
        callback(this.version())
        return () => {}
    }

    listenForHoldings(callback: (change: Counts) => void): () => void {
        this.holdingsListeners.add(callback)
        return () => this.holdingsListeners.delete(callback)
    }

    listenForFortifications(): () => void {
        return () => {}
    }

    private codeOf(index: number): string | undefined {
        return index === 0 ? undefined : this.reel.codes[index] || undefined
    }

    private version(): MapVersion {
        return {epoch: EPOCH, seq: this.seeks}
    }
}
