export type InfectionChange = {
    tile: number
    // Unset when the tile is infected no more.
    virus: string | undefined
}

// The tiles to mark and unmark for a list of changes. A tile may change twice in one event, taken then cured by the
// virus's end: its last change is what it is.
export function infectionMarksOf(changes: readonly InfectionChange[]): {on: number[], off: number[]} {
    const last = new Map(changes.map(({tile, virus}) => [tile, virus]))
    const on: number[] = []
    const off: number[] = []
    for (const [tile, virus] of last) (virus === undefined ? off : on).push(tile)
    return {on, off}
}

// The virus that infects each tile, as the server says. A change of owner cures a tile, as on the server.
export class TileInfections {
    private readonly viruses = new Map<number, string>()

    constructor(public readonly size: number) {}

    // How many tiles a live virus infects.
    public count(): number {
        return this.viruses.size
    }

    public virusOf(tile: number): string | undefined {
        return this.viruses.get(tile)
    }

    public set(tile: number, virus: string | undefined, changes: InfectionChange[]): void {
        if (!Number.isInteger(tile) || tile < 1 || tile > this.size) return
        if (this.viruses.get(tile) === virus) return
        if (virus === undefined) this.viruses.delete(tile)
        else this.viruses.set(tile, virus)
        changes.push({tile, virus})
    }

    // Every tile the virus infects: it ended.
    public tilesOf(virus: string): number[] {
        const tiles: number[] = []
        for (const [tile, infecting] of this.viruses) if (infecting === virus) tiles.push(tile)
        return tiles
    }
}
