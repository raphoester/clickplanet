export type OwnerChange = {
    tile: number
    country: string | undefined
}

export type OptimisticClaim = {
    readonly tile: number
    readonly token: number
}

export type OptimisticPaint = {
    changes: OwnerChange[]
    claim: OptimisticClaim | undefined
}

type Pending = {
    readonly inFlight: Map<number, string>
    country: string | undefined
}

// The flag on each tile, and the player's own clicks painted before the server agreed to them.
export class TileOwnership {
    private readonly owners: (string | undefined)[]
    private readonly pending = new Map<number, Pending>()
    private lastToken = 0

    constructor(public readonly size: number) {
        this.owners = new Array<string | undefined>(size + 1)
    }

    public ownerOf(tile: number): string | undefined {
        return this.owners[tile]
    }

    // From a read: a click in flight stays painted, and the read is what its rollback goes back to.
    public take(tile: number, country: string | undefined, changes: OwnerChange[]): void {
        if (!this.inRange(tile)) return
        const pending = this.pending.get(tile)
        if (pending) {
            pending.country = country
            return
        }
        if (this.assign(tile, country)) changes.push({tile, country})
    }

    // From the stream: what the server did wins over a click in flight, the click's own echo included.
    public set(tile: number, country: string | undefined, changes: OwnerChange[]): void {
        if (!this.inRange(tile)) return
        this.pending.delete(tile)
        if (this.assign(tile, country)) changes.push({tile, country})
    }

    public applyOptimistic(tile: number, country: string): OptimisticPaint {
        if (!this.inRange(tile)) return {changes: [], claim: undefined}

        const token = ++this.lastToken
        const pending = this.pending.get(tile)
        if (pending) {
            pending.inFlight.set(token, country)
        } else {
            this.pending.set(tile, {inFlight: new Map([[token, country]]), country: this.owners[tile]})
        }

        const changes = this.assign(tile, country) ? [{tile, country}] : []
        return {changes, claim: {tile, token}}
    }

    public rollback(claim: OptimisticClaim | undefined): OwnerChange[] {
        if (!claim) return []

        const pending = this.pending.get(claim.tile)
        if (!pending || !pending.inFlight.delete(claim.token)) return []

        const newest = last(pending.inFlight.values())
        if (newest !== undefined) return this.change(claim.tile, newest)

        this.pending.delete(claim.tile)
        return this.change(claim.tile, pending.country)
    }

    private change(tile: number, country: string | undefined): OwnerChange[] {
        return this.assign(tile, country) ? [{tile, country}] : []
    }

    private assign(tile: number, country: string | undefined): boolean {
        if (this.owners[tile] === country) return false
        this.owners[tile] = country
        return true
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
