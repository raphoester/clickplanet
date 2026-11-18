import type {Counts} from "../backends/backend.ts"

export type Recount = {
    countries: boolean
    landmasses: number[]
    provinces: number[]
}

type CountryRow = {tiles: number, seq: number}

type GroundRow = {flags: ReadonlyMap<string, number>, seq: number}

export type Leader = {holder: string, held: number}

const NOBODY: ReadonlyMap<string, number> = new Map()

// What each flag holds of the map, of each landmass and of each province, as the server counts it. A row takes only what
// is newer than it.
export class Holdings {
    // A row at 0 is kept with its version, so an older copy cannot bring it back.
    private readonly countries = new Map<string, CountryRow>()
    private readonly landmasses = new GroundRows()
    private readonly provinces = new GroundRows()

    public read(read: Counts): Recount {
        return this.take(read, true)
    }

    public change(change: Counts): Recount {
        return this.take(change, false)
    }

    // The server started again: no version held compares with its, so the next read replaces every row.
    public forget(): void {
        for (const row of this.countries.values()) row.seq = -1
        this.landmasses.forget()
        this.provinces.forget()
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
        return this.landmasses.holdingsOf(landmass)
    }

    // Whoever holds the most of it, however little; ties on the flag's code, as on the board.
    public leaderOf(landmass: number): Leader | undefined {
        return this.landmasses.leaderOf(landmass)
    }

    public holdingsOfProvince(province: number): ReadonlyMap<string, number> {
        return this.provinces.holdingsOf(province)
    }

    public leaderOfProvince(province: number): Leader | undefined {
        return this.provinces.leaderOf(province)
    }

    private take({version: {seq}, countries, landmasses, provinces}: Counts, whole: boolean): Recount {
        let countriesMoved = false
        countries.forEach((tiles, flag) => {
            if (this.setCountry(flag, tiles, seq)) countriesMoved = true
        })
        if (whole) {
            for (const flag of [...this.countries.keys()]) {
                if (!countries.has(flag) && this.setCountry(flag, 0, seq)) countriesMoved = true
            }
        }
        return {
            countries: countriesMoved,
            landmasses: this.landmasses.take(landmasses, seq, whole),
            provinces: this.provinces.take(provinces, seq, whole),
        }
    }

    private setCountry(flag: string, tiles: number, seq: number): boolean {
        const row = this.countries.get(flag)
        if (row && row.seq >= seq) return false
        this.countries.set(flag, {tiles, seq})
        return (row?.tiles ?? 0) !== tiles
    }
}

// The flags of each ground, landmass or province, each row whole.
class GroundRows {
    private readonly rows = new Map<number, GroundRow>()

    forget(): void {
        for (const row of this.rows.values()) row.seq = -1
    }

    holdingsOf(ground: number): ReadonlyMap<string, number> {
        return this.rows.get(ground)?.flags ?? NOBODY
    }

    leaderOf(ground: number): Leader | undefined {
        let leader: Leader | undefined
        for (const [holder, held] of this.holdingsOf(ground)) {
            if (!leader || held > leader.held || (held === leader.held && holder < leader.holder)) leader = {holder, held}
        }
        return leader
    }

    // The grounds whose row moved.
    take(rows: ReadonlyMap<number, ReadonlyMap<string, number>>, seq: number, whole: boolean): number[] {
        const moved: number[] = []
        rows.forEach((flags, ground) => {
            if (this.set(ground, flags, seq)) moved.push(ground)
        })
        if (whole) {
            for (const ground of [...this.rows.keys()]) {
                if (!rows.has(ground) && this.set(ground, NOBODY, seq)) moved.push(ground)
            }
        }
        return moved
    }

    private set(ground: number, flags: ReadonlyMap<string, number>, seq: number): boolean {
        const row = this.rows.get(ground)
        if (row && row.seq >= seq) return false
        const held = new Map<string, number>()
        flags.forEach((tiles, flag) => {
            if (tiles > 0) held.set(flag, tiles)
        })
        this.rows.set(ground, {flags: held, seq})
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
