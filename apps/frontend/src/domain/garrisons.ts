import type {Update} from "../backends/backend.ts"

export type GarrisonChange = {
    tile: number
    defenders: number
    was: number
}

export type ClickOutcome = "taken" | "defended" | "unchanged"

export function outcomeOf(owner: string | undefined, flag: string, defenders: number): ClickOutcome {
    if (owner === flag) return "unchanged"
    return defenders > 0 ? "defended" : "taken"
}

export type Placement = "place" | "full" | "click"

export function placementOf(owner: string | undefined, flag: string, defenders: number, most: number | undefined): Placement {
    if (owner !== flag) return "click"
    return most !== undefined && most > 0 && defenders >= most ? "full" : "place"
}

export class TileGarrisons {
    private readonly counts: Uint16Array
    private readonly live: Uint8Array

    constructor(public readonly size: number) {
        this.counts = new Uint16Array(size + 1)
        this.live = new Uint8Array(size + 1)
    }

    public defendersOf(tile: number): number {
        return this.inRange(tile) ? this.counts[tile] : 0
    }

    public applyBatch(defenders: ReadonlyMap<number, number>): GarrisonChange[] {
        const changes: GarrisonChange[] = []
        defenders.forEach((count, tile) => {
            if (!this.inRange(tile) || this.live[tile]) return
            this.set(tile, count, changes)
        })
        return changes
    }

    public applyUpdates(updates: readonly Update[]): GarrisonChange[] {
        const changes: GarrisonChange[] = []
        for (const {tile, defenders} of updates) this.setLive(tile, defenders, changes)
        return changes
    }

    public applyClears(tiles: readonly number[]): GarrisonChange[] {
        const changes: GarrisonChange[] = []
        for (const tile of tiles) this.setLive(tile, 0, changes)
        return changes
    }

    public applyStrikes(tiles: readonly number[]): GarrisonChange[] {
        const changes: GarrisonChange[] = []
        for (const tile of tiles) this.setLive(tile, this.defendersOf(tile) - 1, changes)
        return changes
    }

    private setLive(tile: number, defenders: number, changes: GarrisonChange[]) {
        if (!this.inRange(tile)) return
        this.live[tile] = 1
        this.set(tile, defenders, changes)
    }

    private set(tile: number, defenders: number, changes: GarrisonChange[]) {
        const was = this.counts[tile]
        const count = Math.max(0, defenders)
        if (count === was) return
        this.counts[tile] = count
        changes.push({tile, defenders: count, was})
    }

    private inRange(tile: number): boolean {
        return Number.isInteger(tile) && tile >= 1 && tile <= this.size
    }
}
