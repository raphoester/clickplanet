// The map every step of a window: the first frame holds every tile held at its start, each next one the tiles whose
// owner changed since the frame before ("" for a tile a bomb left empty).
export type MapFrame = {at: number, tiles: Uint32Array, owners: string[]}

export type MapHistory = {
    frames: MapFrame[]
    // Each tile's changes of owner and shield hits in the window.
    hits: Map<number, number>
}

const CHECKPOINT_EVERY = 32

// Any frame's map, from a copy kept every few frames: a seek back costs a copy and a few frames, never the whole replay.
export class MapReel {
    readonly codes: string[] = [""]
    private readonly indexes = new Map<string, number>([["", 0]])
    private readonly checkpoints: Uint16Array[] = []

    constructor(private readonly history: MapHistory, private readonly tilesCount: number) {
        const owners = new Uint16Array(tilesCount + 1)
        history.frames.forEach((frame, i) => {
            this.apply(owners, frame)
            if (i % CHECKPOINT_EVERY === 0) this.checkpoints.push(owners.slice())
        })
    }

    get length(): number {
        return this.history.frames.length
    }

    atOf(frame: number): number {
        return this.history.frames[frame]?.at ?? 0
    }

    // Each tile's owner at a frame, as an index in codes; 0 for none.
    ownersAt(frame: number): Uint16Array {
        const at = Math.max(0, Math.min(this.length - 1, frame))
        const kept = Math.floor(at / CHECKPOINT_EVERY)
        const owners = (this.checkpoints[kept] ?? new Uint16Array(this.tilesCount + 1)).slice()
        for (let i = kept * CHECKPOINT_EVERY + 1; i <= at; i++) this.apply(owners, this.history.frames[i])
        return owners
    }

    // The tiles each flag holds at a frame, most first.
    holdingsAt(frame: number): {code: string, tiles: number}[] {
        const counts = new Map<number, number>()
        for (const owner of this.ownersAt(frame)) if (owner !== 0) counts.set(owner, (counts.get(owner) ?? 0) + 1)
        return [...counts].map(([index, tiles]) => ({code: this.codes[index], tiles})).sort((a, b) => b.tiles - a.tiles)
    }

    private apply(owners: Uint16Array, frame: MapFrame) {
        frame.tiles.forEach((tile, i) => {
            if (tile <= this.tilesCount) owners[tile] = this.indexOf(frame.owners[i])
        })
    }

    private indexOf(code: string): number {
        let index = this.indexes.get(code)
        if (index === undefined) {
            index = this.codes.length
            this.codes.push(code)
            this.indexes.set(code, index)
        }
        return index
    }
}
