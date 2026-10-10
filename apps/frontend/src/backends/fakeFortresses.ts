import {Landmasses} from "../domain/landmasses.ts"

// The server's fortify rule for FakeBackend: a whole landmass is fortified, never twice in a row by one flag.
export class FakeFortresses {
    private readonly landmasses: Landmasses
    private readonly held: Map<string, number>[]
    private readonly locks = new Map<number, string>()

    constructor(private readonly assignment: Uint16Array, count: number, ownerOf: (tile: number) => string | undefined) {
        this.landmasses = new Landmasses(assignment, count)
        this.held = Array.from({length: count}, () => new Map<string, number>())

        assignment.forEach((_, i) => this.moved(i + 1, undefined, ownerOf(i + 1)))
        for (let landmass = 1; landmass < count; landmass++) {
            const tiles = this.landmasses.tilesOf(landmass)
            const holder = tiles.length > 0 ? ownerOf(tiles[0]) : undefined
            if (holder !== undefined && this.whole(landmass, holder)) this.locks.set(landmass, holder)
        }
    }

    public moved(tile: number, from: string | undefined, to: string | undefined) {
        const landmass = this.assignment[tile - 1]
        if (!landmass) return
        const held = this.held[landmass]
        if (from !== undefined) held.set(from, (held.get(from) ?? 1) - 1)
        if (to !== undefined) held.set(to, (held.get(to) ?? 0) + 1)
    }

    // The landmass the take of this tile fortifies, if any; the lock moves to the flag.
    public fortify(tile: number, flag: string): {landmass: number, tiles: Uint32Array} | undefined {
        const landmass = this.assignment[tile - 1]
        if (!landmass || this.locks.get(landmass) === flag || !this.whole(landmass, flag)) return undefined
        this.locks.set(landmass, flag)
        return {landmass, tiles: this.landmasses.tilesOf(landmass)}
    }

    public tilesOf(landmass: number): Uint32Array {
        return this.landmasses.tilesOf(landmass)
    }

    public fortresses(): Map<number, string> {
        return new Map(this.locks)
    }

    private whole(landmass: number, flag: string): boolean {
        return (this.held[landmass].get(flag) ?? 0) === this.landmasses.tilesOf(landmass).length
    }
}
