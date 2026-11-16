export type MineChange = {
    tile: number
    mined: boolean
}

export type Laying = "lay" | "refused" | "click"

// What a tap does while a mine is aimed: a tile of another flag, or of none, is clicked first, as with shield on; a tile
// of the flag takes the mine unless one is on it already, or it is a capital whose take would knock the flag out.
export function layingOf(owner: string | undefined, flag: string, mined: boolean, knockOutCapital: boolean): Laying {
    if (owner !== flag) return "click"
    return mined || knockOutCapital ? "refused" : "lay"
}

// The tiles to mark and unmark for a list of changes: a tile's last change is what it is.
export function mineMarksOf(changes: readonly MineChange[]): {on: number[], off: number[]} {
    const last = new Map(changes.map(({tile, mined}) => [tile, mined]))
    const on: number[] = []
    const off: number[] = []
    for (const [tile, mined] of last) (mined ? on : off).push(tile)
    return {on, off}
}

// The mines of the page's flag, each with the map's seq it is known at: a change of its tile after it takes it off, as on
// the server, and a word older than it changes nothing.
export class TileMines {
    private readonly mines = new Map<number, number>()

    constructor(public readonly size: number) {}

    public count(): number {
        return this.mines.size
    }

    public has(tile: number): boolean {
        return this.mines.has(tile)
    }

    public tiles(): number[] {
        return [...this.mines.keys()]
    }

    public lay(tile: number, seq: number, changes: MineChange[]): void {
        if (!Number.isInteger(tile) || tile < 1 || tile > this.size) return
        const known = this.mines.get(tile)
        if (known !== undefined && known >= seq) return
        this.mines.set(tile, seq)
        if (known === undefined) changes.push({tile, mined: true})
    }

    // The tile changed at seq: a mine known before it is gone.
    public takeOff(tile: number, seq: number, changes: MineChange[]): void {
        const known = this.mines.get(tile)
        if (known === undefined || known >= seq) return
        this.mines.delete(tile)
        changes.push({tile, mined: false})
    }

    public forget(changes: MineChange[]): void {
        for (const tile of this.mines.keys()) changes.push({tile, mined: false})
        this.mines.clear()
    }
}
