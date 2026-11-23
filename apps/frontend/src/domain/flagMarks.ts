export type MarkChange = {
    tile: number
    marked: boolean
}

// The tiles to mark and unmark for a list of changes: a tile's last change is what it is.
export function marksOf(changes: readonly MarkChange[]): {on: number[], off: number[]} {
    const last = new Map(changes.map(({tile, marked}) => [tile, marked]))
    const on: number[] = []
    const off: number[] = []
    for (const [tile, marked] of last) (marked ? on : off).push(tile)
    return {on, off}
}

// The mines, or the traps, of the page's flag, each with the map's seq it is known at: a change of its tile after it takes
// it off, as on the server, and a word older than it changes nothing.
export class FlagMarks {
    private readonly marks = new Map<number, number>()

    constructor(public readonly size: number) {}

    public count(): number {
        return this.marks.size
    }

    public has(tile: number): boolean {
        return this.marks.has(tile)
    }

    public tiles(): number[] {
        return [...this.marks.keys()]
    }

    public lay(tile: number, seq: number, changes: MarkChange[]): void {
        if (!Number.isInteger(tile) || tile < 1 || tile > this.size) return
        const known = this.marks.get(tile)
        if (known !== undefined && known >= seq) return
        this.marks.set(tile, seq)
        if (known === undefined) changes.push({tile, marked: true})
    }

    // The tile changed at seq: a mark known before it is gone.
    public takeOff(tile: number, seq: number, changes: MarkChange[]): void {
        const known = this.marks.get(tile)
        if (known === undefined || known >= seq) return
        this.marks.delete(tile)
        changes.push({tile, marked: false})
    }

    public forget(changes: MarkChange[]): void {
        for (const tile of this.marks.keys()) changes.push({tile, marked: false})
        this.marks.clear()
    }
}
