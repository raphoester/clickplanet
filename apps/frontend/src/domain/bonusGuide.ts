import {ChargeKind} from "./bonus.ts"

export const BONUS_GUIDE_STORAGE_KEY = "clickplanet-bonus-guide"

export const LEARNING_USES = 3

const KINDS: readonly ChargeKind[] = ["refill", "bomb", "spreadClicks", "encloseClicks", "shields"]

export type BonusGuide = {
    won: readonly ChargeKind[]
    uses: Readonly<Partial<Record<ChargeKind, number>>>
}

export const FRESH_GUIDE: BonusGuide = {won: [], uses: {}}

export function parseBonusGuide(raw: string | null): BonusGuide {
    if (raw === null) return FRESH_GUIDE

    let stored: unknown
    try {
        stored = JSON.parse(raw)
    } catch {
        return FRESH_GUIDE
    }
    if (typeof stored !== "object" || stored === null) return FRESH_GUIDE

    const {won, uses} = stored as {won?: unknown, uses?: unknown}
    return {
        won: Array.isArray(won) ? KINDS.filter(kind => won.includes(kind)) : [],
        uses: usesFrom(uses),
    }
}

function usesFrom(raw: unknown): Partial<Record<ChargeKind, number>> {
    if (typeof raw !== "object" || raw === null) return {}

    const uses: Partial<Record<ChargeKind, number>> = {}
    for (const kind of KINDS) {
        const count = (raw as Record<string, unknown>)[kind]
        if (typeof count === "number" && Number.isInteger(count) && count > 0) uses[kind] = Math.min(count, LEARNING_USES)
    }
    return uses
}

export function usesOf(guide: BonusGuide, kind: ChargeKind): number {
    return guide.uses[kind] ?? 0
}

export function isNew(guide: BonusGuide, kind: ChargeKind): boolean {
    return usesOf(guide, kind) === 0
}

export function isLearning(guide: BonusGuide, kind: ChargeKind): boolean {
    return usesOf(guide, kind) < LEARNING_USES
}

export function isFirstWin(guide: BonusGuide, kind: ChargeKind): boolean {
    return !guide.won.includes(kind)
}

export function afterUse(guide: BonusGuide, kind: ChargeKind): BonusGuide {
    const uses = usesOf(guide, kind)
    if (uses >= LEARNING_USES) return guide
    return {...guide, uses: {...guide.uses, [kind]: uses + 1}}
}

export function afterWin(guide: BonusGuide, kind: ChargeKind): BonusGuide {
    if (guide.won.includes(kind)) return guide
    return {...guide, won: [...guide.won, kind]}
}
