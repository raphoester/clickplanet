import type {Ownerships, Update} from "../backends/backend.ts";

/** A tile whose owner changed, in the form the renderer needs to draw it. */
export type OwnerChange = {
    tile: number
    country: string
}

/**
 * The authoritative tile -> country map, and the per-country counts derived
 * from it.
 *
 * Two sources write here and they disagree more often than you would think.
 * The initial ownership load arrives as ~26 sequential batches over several
 * seconds, and live updates stream in the whole time. A live update for a tile
 * whose batch had not landed yet used to be overwritten by that batch's stale
 * value, and counted twice in the leaderboard on top of that.
 *
 * So live updates win permanently: a tile someone has claimed is never taken
 * back by a batch that was already in flight when they claimed it. Counts are
 * maintained from the map rather than from what the server said the previous
 * owner was, which is what let them drift below zero.
 */
export class TileOwnership {
    /** Indexed by tile id, which is 1-based; slot 0 is unused. */
    private readonly owners: (string | undefined)[]
    private readonly claimedLive: Uint8Array
    private readonly countsByCountry = new Map<string, number>()

    constructor(public readonly size: number) {
        this.owners = new Array<string | undefined>(size + 1)
        this.claimedLive = new Uint8Array(size + 1)
    }

    public ownerOf(tile: number): string | undefined {
        return this.owners[tile]
    }

    public counts(): ReadonlyMap<string, number> {
        return this.countsByCountry
    }

    /**
     * Applies one batch of the initial load, yielding only the tiles that
     * actually changed so the renderer can skip the rest.
     */
    public applyBatch(ownerships: Ownerships): OwnerChange[] {
        const changes: OwnerChange[] = []
        ownerships.bindings.forEach((country, tile) => {
            if (!this.inRange(tile) || this.claimedLive[tile]) return
            if (this.assign(tile, country)) changes.push({tile, country})
        })
        return changes
    }

    /** Applies live updates, which always win over the initial load. */
    public applyUpdates(updates: Update[]): OwnerChange[] {
        const changes: OwnerChange[] = []
        for (const update of updates) {
            const {tile, newCountry: country} = update
            if (!this.inRange(tile)) continue

            this.claimedLive[tile] = 1
            if (this.assign(tile, country)) changes.push({tile, country})
        }
        return changes
    }

    /** Returns whether the owner actually changed. */
    private assign(tile: number, country: string): boolean {
        const previous = this.owners[tile]
        if (previous === country) return false

        if (previous !== undefined) this.decrement(previous)
        this.owners[tile] = country
        this.countsByCountry.set(country, (this.countsByCountry.get(country) ?? 0) + 1)
        return true
    }

    private decrement(country: string) {
        const remaining = (this.countsByCountry.get(country) ?? 0) - 1
        if (remaining > 0) {
            this.countsByCountry.set(country, remaining)
        } else {
            this.countsByCountry.delete(country)
        }
    }

    /**
     * The tile map and the server's idea of it are generated separately, so an
     * id outside the geometry is possible and must not corrupt the counts.
     */
    private inRange(tile: number): boolean {
        return Number.isInteger(tile) && tile >= 1 && tile <= this.size
    }
}
