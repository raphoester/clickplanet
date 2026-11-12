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

export type Placement = "place" | "full" | "capital" | "click"

export function placementOf(owner: string | undefined, flag: string, shields: number, most: number | undefined, capital = false): Placement {
    if (owner !== flag) return "click"
    if (capital) return "capital"
    return most !== undefined && most > 0 && shields >= most ? "full" : "place"
}

export class TileShields {
    private readonly counts: Uint16Array

    constructor(public readonly size: number) {
        this.counts = new Uint16Array(size + 1)
    }

    public shieldsOf(tile: number): number {
        return this.inRange(tile) ? this.counts[tile] : 0
    }

    // The most shields any tile holds.
    public most(): number {
        let most = 0
        for (let tile = 1; tile <= this.size; tile++) if (this.counts[tile] > most) most = this.counts[tile]
        return most
    }

    public set(tile: number, shields: number, changes: ShieldChange[]): void {
        if (!this.inRange(tile)) return
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
