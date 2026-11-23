import {type Cell, CellPacking, cellSize, flagWidthAt, LEVELS, levelFor, type SlotBudget, unitsOf} from "./slotCells.ts"

export const ATLAS_MARGIN = 1.25

export const EVICTION_MARGIN = 1.2

// Texels per pixel of land on screen: a frame can be far larger than its land, as Chile's.
const SHARES = [16, 8, 4, 2, 1, 0.5, 0.25]

export type FlagNeed = {
    need: number
    area: number
    aspect: number
    spriteWidth: number
}

// A flag needed on several grounds: as wide as its widest need, with all its land.
export function addNeed(needs: Map<string, FlagNeed>, code: string, need: FlagNeed) {
    const was = needs.get(code)
    needs.set(code, was === undefined ? need : {...need, need: Math.max(was.need, need.need), area: was.area + need.area})
}

export function joinedNeeds(a: ReadonlyMap<string, FlagNeed>, b: ReadonlyMap<string, FlagNeed>): Map<string, FlagNeed> {
    const joined = new Map(a)
    for (const [code, need] of b) addNeed(joined, code, need)
    return joined
}

export type Ranked = {code: string, level: number, least: number}

type Candidate = {code: string, flag: FlagNeed, ideal: number, least: number}

export function rankSlots(needs: ReadonlyMap<string, FlagNeed>, held: ReadonlyMap<string, number>, budget: SlotBudget): Ranked[] {
    const room = budget.roots * unitsOf(0)
    const inView: Candidate[] = []
    const kept: (Ranked & {priority: number})[] = []
    for (const [code, flag] of needs) {
        const useful = (level: number) => flagWidthAt(level, flag.aspect) >= flag.spriteWidth * ATLAS_MARGIN
        let least = LEVELS - 1
        while (least > 0 && !useful(least)) least--
        if (flag.need >= flag.spriteWidth * ATLAS_MARGIN) inView.push({code, flag, ideal: levelFor(flag.need, flag.aspect, budget.largest), least})
    }
    for (const [code, level] of held) {
        if (!inView.some((candidate) => candidate.code === code)) kept.push({code, level, least: level, priority: 0})
    }

    const unitsAt = (candidate: Candidate, texels: number) => {
        const level = levelAtShare(candidate, texels)
        return level > candidate.least ? 0 : unitsOf(level)
    }
    const fitting = SHARES.findIndex((texels) => inView.reduce((units, candidate) => units + unitsAt(candidate, texels), 0) <= room)
    const share = fitting < 0 ? SHARES[SHARES.length - 1] : SHARES[fitting]
    const larger = fitting > 0 ? SHARES[fitting - 1] : share
    const targets = new Map(inView.map((candidate) => [candidate.code, share]))
    let spare = room - inView.reduce((units, candidate) => units + unitsAt(candidate, share), 0)
    for (const candidate of [...inView].sort((a, b) => b.flag.area - a.flag.area || a.code.localeCompare(b.code))) {
        const extra = unitsAt(candidate, larger) - unitsAt(candidate, share)
        if (extra <= 0 || extra > spare) continue
        targets.set(candidate.code, larger)
        spare -= extra
    }

    const ranked: (Ranked & {priority: number})[] = [...kept]
    for (const candidate of inView) {
        const {code, flag, least} = candidate
        const target = levelAtShare(candidate, targets.get(code)!)
        const current = held.get(code)
        if (current === undefined) {
            if (target <= least) ranked.push({code, level: target, least, priority: flag.area})
            continue
        }
        const grows = target < current && flag.need > flagWidthAt(current, flag.aspect) * EVICTION_MARGIN
        const level = grows || Math.abs(target - current) > 1 ? Math.min(target, least) : Math.max(current, budget.largest)
        ranked.push({code, level, least, priority: flag.area * EVICTION_MARGIN})
    }

    ranked.sort((a, b) => b.priority - a.priority || a.code.localeCompare(b.code))
    let left = room
    const placed: Ranked[] = []
    for (const {code, level, least} of ranked) {
        let at = level
        while (at < least && unitsOf(at) > left) at++
        if (unitsOf(at) > left) continue
        left -= unitsOf(at)
        placed.push({code, level: at, least})
    }
    return placed
}

function levelAtShare({flag, ideal}: Candidate, texels: number): number {
    let capped = ideal
    while (capped < LEVELS - 1 && cellArea(capped) > flag.area * texels) capped++
    return capped
}

function cellArea(level: number): number {
    const {width, height} = cellSize(level)
    return width * height
}

export type Placed = {code: string, cell: Cell}

export class SlotLayout {
    private readonly packing: CellPacking
    private readonly placed = new Map<string, Cell>()
    private readonly levels = new Map<string, number>()

    constructor(readonly budget: SlotBudget) {
        this.packing = new CellPacking(budget.roots)
    }

    arrange(needs: ReadonlyMap<string, FlagNeed>): Placed[] {
        const ranked = rankSlots(needs, this.levels, this.budget)
        const kept = new Map(ranked.map((rank) => [rank.code, rank]))
        for (const [code, level] of this.levels) {
            if (kept.get(code)?.level === level) continue
            this.forget(code)
        }

        const waiting = ranked.filter(({code}) => !this.placed.has(code)).sort((a, b) => a.level - b.level)
        for (const {code, level, least} of waiting) {
            for (let at = level; at <= least; at++) {
                const cell = this.packing.place(at)
                if (cell === undefined) continue
                this.placed.set(code, cell)
                this.levels.set(code, at)
                break
            }
        }
        return ranked.filter(({code}) => this.placed.has(code)).map(({code}) => ({code, cell: this.placed.get(code)!}))
    }

    forget(code: string) {
        const cell = this.placed.get(code)
        if (cell === undefined) return
        this.packing.release(cell)
        this.placed.delete(code)
        this.levels.delete(code)
    }
}
