import type {Ownerships, Update} from "../backends/backend.ts";

export type OwnerChange = {
    tile: number
    // undefined means the tile went back to unowned
    country: string | undefined
}

// The receipt for an optimistic paint, handed back to `rollback` if the server
// refuses the click it was betting on.
export type OptimisticClaim = {
    readonly tile: number
    readonly token: number
}

export type OptimisticPaint = {
    changes: OwnerChange[]
    claim: OptimisticClaim | undefined
}

type Pending = {
    // the claims still in flight on this tile, oldest first, each with what it painted
    readonly inFlight: Map<number, string>
    // what the tile would hold had none of them happened
    country: string | undefined
    claimedLive: boolean
}

export class TileOwnership {
    private readonly owners: (string | undefined)[]
    private readonly claimedLive: Uint8Array
    private readonly countsByCountry = new Map<string, number>()
    private readonly pending = new Map<number, Pending>()
    private lastToken = 0

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

    public applyBatch(ownerships: Ownerships): OwnerChange[] {
        const changes: OwnerChange[] = []
        ownerships.bindings.forEach((country, tile) => {
            if (!this.inRange(tile)) return

            // A tile with a click in flight keeps its optimistic paint, but the
            // batch is what it falls back to if that click is refused.
            const pending = this.pending.get(tile)
            if (pending) {
                if (!pending.claimedLive) pending.country = country
                return
            }

            if (this.claimedLive[tile]) return
            if (this.assign(tile, country)) changes.push({tile, country})
        })
        return changes
    }

    public applyUpdates(updates: Update[]): OwnerChange[] {
        const changes: OwnerChange[] = []
        for (const update of updates) {
            const {tile, newCountry: country} = update
            if (!this.inRange(tile)) continue

            // The server has spoken about this tile, so nothing in flight on it
            // is worth rolling back to any more.
            this.pending.delete(tile)

            this.claimedLive[tile] = 1
            if (this.assign(tile, country)) changes.push({tile, country})
        }
        return changes
    }

    // Paints a click before the server has agreed to it, remembering enough to
    // take it back.
    public applyOptimistic(tile: number, country: string): OptimisticPaint {
        if (!this.inRange(tile)) return {changes: [], claim: undefined}

        const token = ++this.lastToken
        const pending = this.pending.get(tile)
        if (pending) {
            pending.inFlight.set(token, country)
        } else {
            this.pending.set(tile, {
                inFlight: new Map([[token, country]]),
                country: this.owners[tile],
                claimedLive: this.claimedLive[tile] === 1,
            })
        }

        this.claimedLive[tile] = 1
        const changes = this.assign(tile, country) ? [{tile, country}] : []
        return {changes, claim: {tile, token}}
    }

    // Takes back a refused click. A no-op once the tile has moved on: the
    // server settled it, or a later click on it was refused first.
    public rollback(claim: OptimisticClaim | undefined): OwnerChange[] {
        if (!claim) return []

        const pending = this.pending.get(claim.tile)
        if (!pending || !pending.inFlight.delete(claim.token)) return []

        // Another click on this tile is still in flight; it owns the paint.
        const newest = last(pending.inFlight.values())
        if (newest !== undefined) return this.change(claim.tile, newest)

        this.pending.delete(claim.tile)
        this.claimedLive[claim.tile] = pending.claimedLive ? 1 : 0
        return this.change(claim.tile, pending.country)
    }

    private change(tile: number, country: string | undefined): OwnerChange[] {
        return this.assign(tile, country) ? [{tile, country}] : []
    }

    private assign(tile: number, country: string | undefined): boolean {
        const previous = this.owners[tile]
        if (previous === country) return false

        if (previous !== undefined) this.decrement(previous)
        this.owners[tile] = country
        if (country !== undefined) {
            this.countsByCountry.set(country, (this.countsByCountry.get(country) ?? 0) + 1)
        }
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

    private inRange(tile: number): boolean {
        return Number.isInteger(tile) && tile >= 1 && tile <= this.size
    }
}

function last<T>(values: Iterable<T>): T | undefined {
    let latest: T | undefined
    for (const value of values) latest = value
    return latest
}
