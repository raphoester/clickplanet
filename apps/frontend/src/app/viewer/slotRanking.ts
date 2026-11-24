import {type Cell, CellPacking, cellSize, flagWidthAt, LEVELS, levelFor, type SlotBudget, unitsOf} from "./slotCells.ts"

export const ATLAS_MARGIN = 1.25

export const EVICTION_MARGIN = 1.2

// Texels per pixel of land on screen: a frame can be far larger than its land, as Chile's.
const SHARES = [16, 8, 4, 2, 1, 0.5, 0.25]

// Blur grows with the stretch of a cell: the least blur over all the land gives a flag texels as its detail to the 2/3.
const DETAIL_POWER = 2 / 3

export type FlagNeed = {
    need: number
    area: number
    aspect: number
    spriteWidth: number
    detail: number
}

function worthOf(flag: FlagNeed): number {
    return flag.area * flag.detail ** DETAIL_POWER
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

type Claim = Ranked & {priority: number, larger?: number}

// Who gets a cell is ranked as if none were held: a held size is kept only where it costs no flag its cell.
export function rankSlots(needs: ReadonlyMap<string, FlagNeed>, held: ReadonlyMap<string, number>, budget: SlotBudget): Ranked[] {
    const room = budget.roots * unitsOf(0)
    const inView: Candidate[] = []
    for (const [code, flag] of needs) {
        const useful = (level: number) => flagWidthAt(level, flag.aspect) >= flag.spriteWidth * ATLAS_MARGIN
        let least = LEVELS - 1
        while (least > 0 && !useful(least)) least--
        if (flag.need >= flag.spriteWidth * ATLAS_MARGIN) inView.push({code, flag, ideal: levelFor(flag.need, flag.aspect, budget.largest), least})
    }

    const targets = targetLevels(inView, room)
    const claims: Claim[] = []
    for (const candidate of inView) {
        const {code, flag, least} = candidate
        const target = targets.get(code)!
        if (target > least) continue
        const current = held.get(code)
        const claim: Claim = {code, level: target, least, priority: worthOf(flag) * (current === undefined ? 1 : EVICTION_MARGIN)}
        if (current !== undefined) {
            const grows = flag.need > flagWidthAt(current, flag.aspect) * EVICTION_MARGIN
            if (current === target + 1 && !grows) claim.level = current
            if (current === target - 1) claim.larger = current
        }
        claims.push(claim)
    }
    claims.sort((a, b) => b.priority - a.priority || a.code.localeCompare(b.code))

    let left = room
    const placed = new Map<string, Ranked>()
    for (const {code, level, least} of claims) {
        let at = level
        while (at < least && unitsOf(at) > left) at++
        if (unitsOf(at) > left) continue
        left -= unitsOf(at)
        placed.set(code, {code, level: at, least})
    }
    for (const {code, level, larger} of claims) {
        const kept = placed.get(code)
        if (larger === undefined || kept?.level !== level) continue
        const extra = unitsOf(larger) - unitsOf(level)
        if (extra > left) continue
        left -= extra
        kept.level = larger
    }
    // A flag out of view, or with too little land in it for a cell of its own, keeps its cell while there is room.
    for (const [code, level] of [...held].sort(([a], [b]) => a.localeCompare(b))) {
        if (placed.has(code) || unitsOf(level) > left) continue
        left -= unitsOf(level)
        placed.set(code, {code, level, least: level})
    }
    return [...placed.values()]
}

function targetLevels(inView: readonly Candidate[], room: number): Map<string, number> {
    const unitsAt = (candidate: Candidate, texels: number) => {
        const level = levelAtShare(candidate, texels)
        return level > candidate.least ? 0 : unitsOf(level)
    }
    const fitting = SHARES.findIndex((texels) => inView.reduce((units, candidate) => units + unitsAt(candidate, texels), 0) <= room)
    const share = fitting < 0 ? SHARES[SHARES.length - 1] : SHARES[fitting]
    const larger = fitting > 0 ? SHARES[fitting - 1] : share
    const shares = new Map(inView.map((candidate) => [candidate.code, share]))
    let spare = room - inView.reduce((units, candidate) => units + unitsAt(candidate, share), 0)
    for (const candidate of [...inView].sort((a, b) => worthOf(b.flag) - worthOf(a.flag) || a.code.localeCompare(b.code))) {
        const extra = unitsAt(candidate, larger) - unitsAt(candidate, share)
        if (extra <= 0 || extra > spare) continue
        shares.set(candidate.code, larger)
        spare -= extra
    }
    return new Map(inView.map((candidate) => [candidate.code, levelAtShare(candidate, shares.get(candidate.code)!)]))
}

function levelAtShare({flag, ideal}: Candidate, texels: number): number {
    let capped = ideal
    while (capped < LEVELS - 1 && cellArea(capped) > worthOf(flag) * texels) capped++
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
