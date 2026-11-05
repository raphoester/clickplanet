import {Landmasses} from "../domain/landmasses.ts"
import {FakeTally} from "./fakeTally.ts"

// The server's fortify rule for FakeBackend: a whole landmass is fortified, never twice in a row by one flag.
export class FakeFortresses {
    private readonly landmasses: Landmasses
    private readonly locks = new Map<number, string>()

    constructor(
        private readonly assignment: Uint16Array,
        count: number,
        private readonly tally: FakeTally,
        ownerOf: (tile: number) => string | undefined,
    ) {
        this.landmasses = new Landmasses(assignment, count)
        for (let landmass = 1; landmass < count; landmass++) {
            const tiles = this.landmasses.tilesOf(landmass)
            const holder = tiles.length > 0 ? ownerOf(tiles[0]) : undefined
            if (holder !== undefined && this.whole(landmass, holder)) this.locks.set(landmass, holder)
        }
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
        return this.tally.heldOf(landmass, flag) === this.landmasses.tilesOf(landmass).length
    }
}
