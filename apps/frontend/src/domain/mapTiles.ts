import type {MapVersion, Ownerships, TileRange, Update} from "../backends/backend.ts"
import {MapClock} from "./mapClock.ts"
import {OwnerChange, TileOwnership} from "./tileOwnership.ts"
import {ShieldChange, TileShields} from "./shields.ts"

export type TileChanges = {
    owners: OwnerChange[]
    shields: ShieldChange[]
}

type Fortifying = {
    tiles: ArrayLike<number>
    flag: string
    seq: number
    most: number
}

const UNKNOWN = -1

// The server's end is inclusive, and a read past either end of the map is what the antibot calls a bot's walk.
export function tileRanges(size: number, batchSize: number): TileRange[] {
    const ranges: TileRange[] = []
    for (let start = 1; start <= size; start += batchSize) {
        ranges.push({start, end: Math.min(start + batchSize - 1, size)})
    }
    return ranges
}

// Every tile's flag and shields. Whatever brings a tile's value, a read or the stream, it is taken only when its version is newer than the tile's.
export class MapTiles {
    readonly ownership: TileOwnership
    readonly shields: TileShields
    readonly ranges: readonly TileRange[]
    private readonly seen: Float64Array
    private readonly reads: (MapVersion | undefined)[]
    // The last version the stream told: it told every change up to it.
    private told: MapVersion | undefined
    // A fortify adds a shield: it waits for the read of a tile whose value may be older than the stream.
    private waiting: Fortifying[] = []

    constructor(size: number, private readonly batchSize: number, private readonly clock: MapClock) {
        this.ownership = new TileOwnership(size)
        this.shields = new TileShields(size)
        this.ranges = tileRanges(size, batchSize)
        this.seen = new Float64Array(size + 1).fill(UNKNOWN)
        this.reads = new Array(this.ranges.length)
    }

    // A new start forgets every version: the tiles keep their flags until they are read again.
    public opened(version: MapVersion): {restarted: boolean} {
        const told = this.told
        if (told && told.epoch === version.epoch) {
            this.reads.forEach((read, index) => {
                if (read && this.current(index) && read.seq < told.seq) this.reads[index] = told
            })
        }
        const restarted = this.clock.opened(version)
        if (restarted) this.seen.fill(UNKNOWN)
        this.told = version
        this.settle()
        return {restarted}
    }

    // A range current when the stream opened stays current up to what the stream told since, as long as it stays open.
    public heard(version: MapVersion): void {
        if (!this.clock.counts(version)) return
        if (!this.told || this.told.epoch !== version.epoch || this.told.seq < version.seq) this.told = version
    }

    public known(tile: number): boolean {
        return this.seen[tile] > UNKNOWN
    }

    // What is older than the stream's opening, or of another start, or never read.
    public stale(ranges: readonly TileRange[]): TileRange[] {
        return ranges.filter(({start}) => {
            const index = this.rangeOf(start)
            return index === undefined || !this.current(index)
        })
    }

    // The share of the ranges read at least once.
    public readShare(): number {
        return this.reads.filter((read) => read !== undefined).length / this.ranges.length
    }

    // Each range read at the stream's opening or after.
    public settled(): boolean {
        return this.ranges.every((_, index) => this.current(index))
    }

    // Whether the range is current: else read it again.
    public read({range, version, bindings, shields}: Ownerships, changes: TileChanges): boolean {
        if (!this.clock.counts(version)) return false
        const index = this.rangeOf(range.start)
        if (index === undefined) return false

        for (let tile = range.start; tile <= range.end; tile++) {
            if (!this.take(tile, version.seq)) continue
            this.ownership.take(tile, bindings.get(tile), changes.owners)
            this.shields.set(tile, shields.get(tile) ?? 0, changes.shields)
        }

        const was = this.reads[index]
        if (!was || was.epoch !== version.epoch || was.seq < version.seq) this.reads[index] = version

        const current = this.current(index)
        if (current) {
            for (const fortifying of this.waiting) {
                if (fortifying.seq > version.seq) this.raise(fortifying, changes.shields, range)
            }
            this.settle()
        }
        return current
    }

    public applyUpdates(updates: readonly Update[], changes: TileChanges): void {
        for (const {tile, newCountry, shields, version} of updates) {
            this.heard(version)
            if (!this.clock.counts(version) || !this.take(tile, version.seq)) continue
            this.ownership.set(tile, newCountry, changes.owners)
            this.shields.set(tile, shields, changes.shields)
        }
    }

    public applyBlast(cleared: readonly number[], struck: readonly {tile: number, shields: number}[], version: MapVersion, changes: TileChanges): void {
        if (!this.clock.counts(version)) return
        for (const tile of cleared) {
            if (!this.take(tile, version.seq)) continue
            this.ownership.set(tile, undefined, changes.owners)
            this.shields.set(tile, 0, changes.shields)
        }
        for (const {tile, shields} of struck) {
            if (this.take(tile, version.seq)) this.shields.set(tile, shields, changes.shields)
        }
    }

    // The server raised every tile of the landmass below the most: its flag held them all.
    public applyFortification(tiles: ArrayLike<number>, flag: string, most: number, version: MapVersion): ShieldChange[] {
        const changes: ShieldChange[] = []
        this.heard(version)
        if (!this.clock.counts(version)) return changes

        const fortifying = {tiles, flag, seq: version.seq, most}
        this.raise(fortifying, changes)
        if (!this.settled()) this.waiting.push(fortifying)
        return changes
    }

    private raise({tiles, flag, seq, most}: Fortifying, changes: ShieldChange[], within?: TileRange) {
        for (let i = 0; i < tiles.length; i++) {
            const tile = tiles[i]
            if (within && (tile < within.start || tile > within.end)) continue
            if (this.seen[tile] >= seq || !this.trusted(tile)) continue
            this.seen[tile] = seq
            const shields = this.shields.shieldsOf(tile)
            if (this.ownership.ownerOf(tile) === flag && shields < most) this.shields.set(tile, shields + 1, changes)
        }
    }

    private take(tile: number, seq: number): boolean {
        if (!(tile >= 1 && tile < this.seen.length) || this.seen[tile] >= seq) return false
        this.seen[tile] = seq
        return true
    }

    private trusted(tile: number): boolean {
        const index = this.rangeOf(tile)
        return this.known(tile) && index !== undefined && this.current(index)
    }

    private current(index: number): boolean {
        const read = this.reads[index]
        return read !== undefined && this.clock.current(read)
    }

    private settle() {
        if (this.waiting.length > 0 && this.settled()) this.waiting = []
    }

    private rangeOf(tile: number): number | undefined {
        const index = Math.floor((tile - 1) / this.batchSize)
        return index >= 0 && index < this.ranges.length ? index : undefined
    }
}
