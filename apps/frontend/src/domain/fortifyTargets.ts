import {Target, targetOf} from "./fortifyTarget.ts"

export type Holdings = {
    sizeOf(province: number): number
    leaderOf(province: number): {holder: string, held: number} | undefined
    ownerOf(tile: number): string | undefined
    known(tile: number): boolean
    tilesOf(province: number): ArrayLike<number>
    provinceOf(tile: number): number
}

// Which provinces a flag is close to fortifying, from the server's counts, and which of their tiles it still misses, from the tiles read.
export class FortifyTargets {
    private readonly locks = new Map<number, string>()
    private readonly targets = new Map<number, Target>()

    constructor(
        private readonly holdings: Holdings,
        private readonly setPulses: (tiles: number[], on: boolean) => void,
    ) {}

    public targetOf(province: number): Target | undefined {
        return this.targets.get(province)
    }

    public lockOf(province: number): string | undefined {
        return this.locks.get(province)
    }

    public provinces(): IterableIterator<number> {
        return this.targets.keys()
    }

    public lockAll(locks: ReadonlyMap<number, string>, provinces: number) {
        this.locks.clear()
        locks.forEach((flag, province) => this.locks.set(province, flag))
        for (let province = 1; province < provinces; province++) this.settle(province, [])
    }

    public lock(province: number, flag: string) {
        this.locks.set(province, flag)
        this.settle(province, [])
    }

    public recount(provinces: readonly number[]) {
        for (const province of provinces) this.settle(province, [])
    }

    public touch(tiles: readonly number[]) {
        const byProvince = new Map<number, number[]>()
        for (const tile of tiles) {
            const province = this.holdings.provinceOf(tile)
            if (!province) continue
            const list = byProvince.get(province)
            if (list) list.push(tile)
            else byProvince.set(province, [tile])
        }

        byProvince.forEach((touched, province) => this.settle(province, touched))
    }

    private settle(province: number, touched: readonly number[]) {
        const leader = this.holdings.leaderOf(province)
        const before = this.targets.get(province)
        const after = targetOf(this.holdings.sizeOf(province), leader?.holder, leader?.held ?? 0, this.locks.get(province))

        if (before && after && before.flag === after.flag) {
            this.targets.set(province, after)
            const on = touched.filter((tile) => this.missing(tile, after.flag))
            const off = touched.filter((tile) => !this.missing(tile, after.flag))
            if (on.length > 0) this.setPulses(on, true)
            if (off.length > 0) this.setPulses(off, false)
            return
        }

        if (before) {
            this.setPulses(Array.from(this.holdings.tilesOf(province)), false)
            this.targets.delete(province)
        }
        if (after) {
            const missing = Array.from(this.holdings.tilesOf(province)).filter((tile) => this.missing(tile, after.flag))
            this.setPulses(missing, true)
            this.targets.set(province, after)
        }
    }

    private missing(tile: number, flag: string): boolean {
        return this.holdings.known(tile) && this.holdings.ownerOf(tile) !== flag
    }
}
