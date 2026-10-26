import type {Counts, MapVersion} from "./backend.ts"

// The server's counts, for the fakes: what each flag holds of the map and of each landmass, and what moved since they were last told.
export class FakeCensus {
    private readonly countries = new Map<string, number>()
    private readonly landmasses: Map<string, number>[]
    private readonly movedCountries = new Set<string>()
    private readonly movedLandmasses = new Set<number>()

    constructor(private readonly assignment: Uint16Array, count: number) {
        this.landmasses = Array.from({length: count}, () => new Map<string, number>())
    }

    public moved(tile: number, from: string | undefined, to: string | undefined): void {
        if (from === to) return
        const landmass = this.assignment[tile - 1] ?? 0
        for (const [flag, by] of [[from, -1], [to, 1]] as const) {
            if (flag === undefined) continue
            add(this.countries, flag, by)
            if (landmass) add(this.landmasses[landmass], flag, by)
            this.movedCountries.add(flag)
        }
        if (landmass) this.movedLandmasses.add(landmass)
    }

    public heldOf(landmass: number, flag: string): number {
        return this.landmasses[landmass]?.get(flag) ?? 0
    }

    public heldBy(flag: string): number {
        return this.countries.get(flag) ?? 0
    }

    public read(version: MapVersion): Counts {
        const landmasses = new Map<number, ReadonlyMap<string, number>>()
        this.landmasses.forEach((row, landmass) => {
            if (row.size > 0) landmasses.set(landmass, new Map(row))
        })
        return {version, countries: new Map(this.countries), landmasses}
    }

    // Every row that moved since the last call, whole, a flag that left the map at 0.
    public changes(version: MapVersion): Counts | undefined {
        if (this.movedCountries.size === 0 && this.movedLandmasses.size === 0) return undefined
        const countries = new Map([...this.movedCountries].map((flag) => [flag, this.heldBy(flag)]))
        const landmasses = new Map([...this.movedLandmasses].map((landmass) => [landmass, new Map(this.landmasses[landmass])]))
        this.movedCountries.clear()
        this.movedLandmasses.clear()
        return {version, countries, landmasses}
    }
}

function add(counts: Map<string, number>, flag: string, by: number) {
    const tiles = (counts.get(flag) ?? 0) + by
    if (tiles > 0) counts.set(flag, tiles)
    else counts.delete(flag)
}
