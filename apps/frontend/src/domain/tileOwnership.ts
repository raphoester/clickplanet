import type {Ownerships, Update} from "../backends/backend.ts";

export type OwnerChange = {
    tile: number
    country: string
}

export class TileOwnership {
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

    public applyBatch(ownerships: Ownerships): OwnerChange[] {
        const changes: OwnerChange[] = []
        ownerships.bindings.forEach((country, tile) => {
            if (!this.inRange(tile) || this.claimedLive[tile]) return
            if (this.assign(tile, country)) changes.push({tile, country})
        })
        return changes
    }

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

    private inRange(tile: number): boolean {
        return Number.isInteger(tile) && tile >= 1 && tile <= this.size
    }
}
