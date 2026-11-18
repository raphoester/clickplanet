import {Landmasses} from "../domain/landmasses.ts"
import {Cut, FakeTally} from "./fakeTally.ts"

// The server's fortify rule for FakeBackend: a whole province is fortified, never twice in a row by one flag.
export class FakeFortresses {
    private readonly provinces: Landmasses
    private readonly locks = new Map<number, string>()

    constructor(
        private readonly cut: Cut,
        private readonly tally: FakeTally,
        ownerOf: (tile: number) => string | undefined,
    ) {
        this.provinces = new Landmasses(cut.assignment, cut.count)
        for (let province = 1; province < cut.count; province++) {
            const tiles = this.provinces.tilesOf(province)
            const holder = tiles.length > 0 ? ownerOf(tiles[0]) : undefined
            if (holder !== undefined && this.whole(province, holder)) this.locks.set(province, holder)
        }
    }

    // The province the take of this tile fortifies, if any; the lock moves to the flag.
    public fortify(tile: number, flag: string): {province: number, tiles: Uint32Array} | undefined {
        const province = this.cut.assignment[tile - 1]
        if (!province || this.locks.get(province) === flag || !this.whole(province, flag)) return undefined
        this.locks.set(province, flag)
        return {province, tiles: this.provinces.tilesOf(province)}
    }

    public tilesOf(province: number): Uint32Array {
        return this.provinces.tilesOf(province)
    }

    public fortresses(): Map<number, string> {
        return new Map(this.locks)
    }

    private whole(province: number, flag: string): boolean {
        return this.tally.heldOfProvince(province, flag) === this.provinces.tilesOf(province).length
    }
}
