import type {Update} from "../backends/backend.ts"

export type ShieldChange = {
    tile: number
    shields: number
    was: number
}

export type ClickOutcome = "taken" | "shielded" | "unchanged"

export function outcomeOf(owner: string | undefined, flag: string, shields: number): ClickOutcome {
    if (owner === flag) return "unchanged"
    return shields > 0 ? "shielded" : "taken"
}

export type Placement = "place" | "full" | "click"

export function placementOf(owner: string | undefined, flag: string, shields: number, most: number | undefined): Placement {
    if (owner !== flag) return "click"
    return most !== undefined && most > 0 && shields >= most ? "full" : "place"
}

export class TileShields {
    private readonly counts: Uint16Array
    private readonly live: Uint8Array

    constructor(public readonly size: number) {
        this.counts = new Uint16Array(size + 1)
        this.live = new Uint8Array(size + 1)
    }

    public shieldsOf(tile: number): number {
        return this.inRange(tile) ? this.counts[tile] : 0
    }

    public applyBatch(shields: ReadonlyMap<number, number>): ShieldChange[] {
        const changes: ShieldChange[] = []
        shields.forEach((count, tile) => {
            if (!this.inRange(tile) || this.live[tile]) return
            this.set(tile, count, changes)
        })
        return changes
    }

    public forgetLive(): void {
        this.live.fill(0)
    }

    public resync(shields: ReadonlyMap<number, number>): ShieldChange[] {
        const changes: ShieldChange[] = []
        for (let tile = 1; tile <= this.size; tile++) {
            if (!this.live[tile]) this.set(tile, shields.get(tile) ?? 0, changes)
        }
        return changes
    }

    public applyUpdates(updates: readonly Update[]): ShieldChange[] {
        const changes: ShieldChange[] = []
        for (const {tile, shields} of updates) this.setLive(tile, shields, changes)
        return changes
    }

    public applyClears(tiles: readonly number[]): ShieldChange[] {
        const changes: ShieldChange[] = []
        for (const tile of tiles) this.setLive(tile, 0, changes)
        return changes
    }

    public applyStrikes(tiles: readonly number[]): ShieldChange[] {
        const changes: ShieldChange[] = []
        for (const tile of tiles) this.setLive(tile, this.shieldsOf(tile) - 1, changes)
        return changes
    }

    // The server raised every tile of the landmass; a tile taken since is not the fortifying flag's.
    public applyFortification(tiles: Iterable<number>, held: (tile: number) => boolean, most: number): ShieldChange[] {
        const changes: ShieldChange[] = []
        for (const tile of tiles) {
            const shields = this.shieldsOf(tile)
            if (held(tile) && shields < most) this.setLive(tile, shields + 1, changes)
        }
        return changes
    }

    private setLive(tile: number, shields: number, changes: ShieldChange[]) {
        if (!this.inRange(tile)) return
        this.live[tile] = 1
        this.set(tile, shields, changes)
    }

    private set(tile: number, shields: number, changes: ShieldChange[]) {
        const was = this.counts[tile]
        const count = Math.max(0, shields)
        if (count === was) return
        this.counts[tile] = count
        changes.push({tile, shields: count, was})
    }

    private inRange(tile: number): boolean {
        return Number.isInteger(tile) && tile >= 1 && tile <= this.size
    }
}
