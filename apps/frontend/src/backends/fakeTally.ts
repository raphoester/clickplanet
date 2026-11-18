import type {Counts, MapVersion} from "./backend.ts"

// A board's tiles, cut into grounds: each tile's, 0 none, and how many there are, 0 included.
export type Cut = {assignment: Uint16Array, count: number}

// The server's counts, for the fakes: what each flag holds of the map, of each landmass and of each province, and what
// moved since they were last told.
export class FakeTally {
    private readonly countries = new Map<string, number>()
    private readonly movedCountries = new Set<string>()
    private readonly landmasses: Grounds
    private readonly provinces: Grounds

    // A board with no provinces file has its landmasses as provinces.
    constructor(landmasses: Cut, provinces: Cut) {
        this.landmasses = new Grounds(landmasses)
        this.provinces = new Grounds(provinces)
    }

    public moved(tile: number, from: string | undefined, to: string | undefined): void {
        if (from === to) return
        for (const [flag, by] of [[from, -1], [to, 1]] as const) {
            if (flag === undefined) continue
            add(this.countries, flag, by)
            this.movedCountries.add(flag)
        }
        this.landmasses.moved(tile, from, to)
        this.provinces.moved(tile, from, to)
    }

    public heldOf(landmass: number, flag: string): number {
        return this.landmasses.heldOf(landmass, flag)
    }

    public heldOfProvince(province: number, flag: string): number {
        return this.provinces.heldOf(province, flag)
    }

    public heldBy(flag: string): number {
        return this.countries.get(flag) ?? 0
    }

    public read(version: MapVersion): Counts {
        return {version, countries: new Map(this.countries), landmasses: this.landmasses.read(), provinces: this.provinces.read()}
    }

    // Every row that moved since the last call, whole, a flag that left the map at 0.
    public changes(version: MapVersion): Counts | undefined {
        if (this.movedCountries.size === 0 && !this.landmasses.anyMoved() && !this.provinces.anyMoved()) return undefined
        const countries = new Map([...this.movedCountries].map((flag) => [flag, this.heldBy(flag)]))
        this.movedCountries.clear()
        return {version, countries, landmasses: this.landmasses.changes(), provinces: this.provinces.changes()}
    }
}

class Grounds {
    private readonly rows: Map<string, number>[]
    private readonly movedRows = new Set<number>()

    constructor(private readonly cut: Cut) {
        this.rows = Array.from({length: cut.count}, () => new Map<string, number>())
    }

    moved(tile: number, from: string | undefined, to: string | undefined): void {
        const ground = this.cut.assignment[tile - 1] ?? 0
        if (!ground) return
        if (from !== undefined) add(this.rows[ground], from, -1)
        if (to !== undefined) add(this.rows[ground], to, 1)
        this.movedRows.add(ground)
    }

    heldOf(ground: number, flag: string): number {
        return this.rows[ground]?.get(flag) ?? 0
    }

    anyMoved(): boolean {
        return this.movedRows.size > 0
    }

    read(): Map<number, ReadonlyMap<string, number>> {
        const rows = new Map<number, ReadonlyMap<string, number>>()
        this.rows.forEach((row, ground) => {
            if (row.size > 0) rows.set(ground, new Map(row))
        })
        return rows
    }

    changes(): Map<number, ReadonlyMap<string, number>> {
        const rows = new Map([...this.movedRows].map((ground) => [ground, new Map(this.rows[ground])] as const))
        this.movedRows.clear()
        return rows
    }
}

function add(counts: Map<string, number>, flag: string, by: number) {
    const tiles = (counts.get(flag) ?? 0) + by
    if (tiles > 0) counts.set(flag, tiles)
    else counts.delete(flag)
}
