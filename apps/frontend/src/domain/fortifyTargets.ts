import {Target, targetOf} from "./fortifyTarget.ts"

export type Holdings = {
    sizeOf(landmass: number): number
    leaderOf(landmass: number): {holder: string, held: number} | undefined
    ownerOf(tile: number): string | undefined
    known(tile: number): boolean
    tilesOf(landmass: number): ArrayLike<number>
    landmassOf(tile: number): number
}

// Which landmasses a flag is close to fortifying, from the server's counts, and which of their tiles it still misses, from the tiles read.
export class FortifyTargets {
    private readonly locks = new Map<number, string>()
    private readonly targets = new Map<number, Target>()

    constructor(
        private readonly holdings: Holdings,
        private readonly setPulses: (tiles: number[], on: boolean) => void,
    ) {}

    public targetOf(landmass: number): Target | undefined {
        return this.targets.get(landmass)
    }

    public lockOf(landmass: number): string | undefined {
        return this.locks.get(landmass)
    }

    public landmasses(): IterableIterator<number> {
        return this.targets.keys()
    }

    public lockAll(locks: ReadonlyMap<number, string>, landmasses: number) {
        this.locks.clear()
        locks.forEach((flag, landmass) => this.locks.set(landmass, flag))
        for (let landmass = 1; landmass < landmasses; landmass++) this.settle(landmass, [])
    }

    public lock(landmass: number, flag: string) {
        this.locks.set(landmass, flag)
        this.settle(landmass, [])
    }

    public recount(landmasses: readonly number[]) {
        for (const landmass of landmasses) this.settle(landmass, [])
    }

    public touch(tiles: readonly number[]) {
        const byLandmass = new Map<number, number[]>()
        for (const tile of tiles) {
            const landmass = this.holdings.landmassOf(tile)
            if (!landmass) continue
            const list = byLandmass.get(landmass)
            if (list) list.push(tile)
            else byLandmass.set(landmass, [tile])
        }

        byLandmass.forEach((touched, landmass) => this.settle(landmass, touched))
    }

    private settle(landmass: number, touched: readonly number[]) {
        const leader = this.holdings.leaderOf(landmass)
        const before = this.targets.get(landmass)
        const after = targetOf(this.holdings.sizeOf(landmass), leader?.holder, leader?.held ?? 0, this.locks.get(landmass))

        if (before && after && before.flag === after.flag) {
            this.targets.set(landmass, after)
            const on = touched.filter((tile) => this.missing(tile, after.flag))
            const off = touched.filter((tile) => !this.missing(tile, after.flag))
            if (on.length > 0) this.setPulses(on, true)
            if (off.length > 0) this.setPulses(off, false)
            return
        }

        if (before) {
            this.setPulses(Array.from(this.holdings.tilesOf(landmass)), false)
            this.targets.delete(landmass)
        }
        if (after) {
            const missing = Array.from(this.holdings.tilesOf(landmass)).filter((tile) => this.missing(tile, after.flag))
            this.setPulses(missing, true)
            this.targets.set(landmass, after)
        }
    }

    private missing(tile: number, flag: string): boolean {
        return this.holdings.known(tile) && this.holdings.ownerOf(tile) !== flag
    }
}
