import type {Counts} from "../backends/backend.ts"

export type Recount = {
    countries: boolean
    landmasses: number[]
}

type CountryRow = {tiles: number, seq: number}

type LandmassRow = {flags: ReadonlyMap<string, number>, seq: number}

const NOBODY: ReadonlyMap<string, number> = new Map()

// What each flag holds of the map and of each landmass, as the server counts it. A row takes only what is newer than it.
export class Holdings {
    // A row at 0 is kept with its version, so an older copy cannot bring it back.
    private readonly countries = new Map<string, CountryRow>()
    private readonly landmasses = new Map<number, LandmassRow>()

    public read(read: Counts): Recount {
        return this.take(read, true)
    }

    public change(change: Counts): Recount {
        return this.take(change, false)
    }

    // The server started again: no version held compares with its, so the next read replaces every row.
    public forget(): void {
        for (const row of this.countries.values()) row.seq = -1
        for (const row of this.landmasses.values()) row.seq = -1
    }

    public counts(): ReadonlyMap<string, number> {
        const counts = new Map<string, number>()
        for (const [flag, {tiles}] of this.countries) {
            if (tiles > 0) counts.set(flag, tiles)
        }
        return counts
    }

    public tilesOf(flag: string): number {
        return this.countries.get(flag)?.tiles ?? 0
    }

    public holdingsOf(landmass: number): ReadonlyMap<string, number> {
        return this.landmasses.get(landmass)?.flags ?? NOBODY
    }

    // Whoever holds the most of it, however little; ties on the flag's code, as on the board.
    public leaderOf(landmass: number): {holder: string, held: number} | undefined {
        let leader: {holder: string, held: number} | undefined
        for (const [holder, held] of this.holdingsOf(landmass)) {
            if (!leader || held > leader.held || (held === leader.held && holder < leader.holder)) leader = {holder, held}
        }
        return leader
    }

    private take({version: {seq}, countries, landmasses}: Counts, whole: boolean): Recount {
        let countriesMoved = false
        const moved: number[] = []

        countries.forEach((tiles, flag) => {
            if (this.setCountry(flag, tiles, seq)) countriesMoved = true
        })
        landmasses.forEach((flags, landmass) => {
            if (this.setLandmass(landmass, flags, seq)) moved.push(landmass)
        })

        if (whole) {
            for (const flag of [...this.countries.keys()]) {
                if (!countries.has(flag) && this.setCountry(flag, 0, seq)) countriesMoved = true
            }
            for (const landmass of [...this.landmasses.keys()]) {
                if (!landmasses.has(landmass) && this.setLandmass(landmass, NOBODY, seq)) moved.push(landmass)
            }
        }

        return {countries: countriesMoved, landmasses: moved}
    }

    private setCountry(flag: string, tiles: number, seq: number): boolean {
        const row = this.countries.get(flag)
        if (row && row.seq >= seq) return false
        this.countries.set(flag, {tiles, seq})
        return (row?.tiles ?? 0) !== tiles
    }

    private setLandmass(landmass: number, flags: ReadonlyMap<string, number>, seq: number): boolean {
        const row = this.landmasses.get(landmass)
        if (row && row.seq >= seq) return false
        const held = new Map<string, number>()
        flags.forEach((tiles, flag) => {
            if (tiles > 0) held.set(flag, tiles)
        })
        this.landmasses.set(landmass, {flags: held, seq})
        return !sameRow(row?.flags ?? NOBODY, held)
    }
}

function sameRow(a: ReadonlyMap<string, number>, b: ReadonlyMap<string, number>): boolean {
    if (a.size !== b.size) return false
    for (const [flag, tiles] of a) {
        if (b.get(flag) !== tiles) return false
    }
    return true
}
