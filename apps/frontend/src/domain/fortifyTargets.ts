import {Target, targetOf} from "./fortifyTarget.ts"

export type Holdings = {
    sizeOf(landmass: number): number
    leaderOf(landmass: number): {holder: string, held: number} | undefined
    ownerOf(tile: number): string | undefined
    tilesOf(landmass: number): ArrayLike<number>
    landmassOf(tile: number): number
}

export type Entered = {landmass: number, target: Target}

// Which landmasses a flag is close to fortifying, and which of their tiles it still misses.
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

    public landmasses(): IterableIterator<number> {
        return this.targets.keys()
    }

    public lockAll(locks: ReadonlyMap<number, string>, landmasses: number): Entered[] {
        this.locks.clear()
        locks.forEach((flag, landmass) => this.locks.set(landmass, flag))
        const entered: Entered[] = []
        for (let landmass = 1; landmass < landmasses; landmass++) this.settle(landmass, [], entered)
        return entered
    }

    public lock(landmass: number, flag: string): Entered[] {
        this.locks.set(landmass, flag)
        const entered: Entered[] = []
        this.settle(landmass, [], entered)
        return entered
    }

    public touch(tiles: readonly number[]): Entered[] {
        const byLandmass = new Map<number, number[]>()
        for (const tile of tiles) {
            const landmass = this.holdings.landmassOf(tile)
            if (!landmass) continue
            const list = byLandmass.get(landmass)
            if (list) list.push(tile)
            else byLandmass.set(landmass, [tile])
        }

        const entered: Entered[] = []
        byLandmass.forEach((touched, landmass) => this.settle(landmass, touched, entered))
        return entered
    }

    private settle(landmass: number, touched: readonly number[], entered: Entered[]) {
        const leader = this.holdings.leaderOf(landmass)
        const before = this.targets.get(landmass)
        const after = targetOf(this.holdings.sizeOf(landmass), leader?.holder, leader?.held ?? 0, this.locks.get(landmass))

        if (before && after && before.flag === after.flag) {
            this.targets.set(landmass, after)
            const on = touched.filter((tile) => this.holdings.ownerOf(tile) !== after.flag)
            const off = touched.filter((tile) => this.holdings.ownerOf(tile) === after.flag)
            if (on.length > 0) this.setPulses(on, true)
            if (off.length > 0) this.setPulses(off, false)
            return
        }

        if (before) {
            this.setPulses(Array.from(this.holdings.tilesOf(landmass)), false)
            this.targets.delete(landmass)
        }
        if (after) {
            const missing = Array.from(this.holdings.tilesOf(landmass)).filter((tile) => this.holdings.ownerOf(tile) !== after.flag)
            this.setPulses(missing, true)
            this.targets.set(landmass, after)
            entered.push({landmass, target: after})
        }
    }
}
