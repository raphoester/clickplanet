import {Teased} from "./battleTeaser.ts"
import {Sale, saleRuns} from "./shop.ts"

const HOUR = 60 * 60_000
const DAY = 24 * HOUR

export const NUDGES_STORAGE_KEY = "clickplanet-nudges"

// What the game asks of a player with no fact behind it, in the order of their first turn.
export const ASKS = ["coffee", "tiktok", "youtube"] as const
export type Ask = typeof ASKS[number]

// The moments due as the map shows come before any nudge.
export const NUDGE_SETTLE_MS = 3_000

// An ask waits this long after the last nudge, the battle's teaser included.
export const ASK_SPACING_MS = 12 * HOUR

const COFFEE_AGAIN_MS = 30 * DAY

// When each ask was last closed, and when its link was last opened.
export type Asked = {
    shown: Partial<Record<Ask, number>>
    opened: Partial<Record<Ask, number>>
}

export const NOTHING_ASKED: Asked = {shown: {}, opened: {}}

export type NudgeScene = {
    guided: boolean
    battlePlays: boolean
}

// While a sale runs its nudge takes the asks' turn: no ask.
export type AskScene = NudgeScene & {
    saleRuns: boolean
}

export function askDue(asked: Asked, teased: Teased | null, scene: AskScene, now: number): Ask | undefined {
    if (scene.guided || scene.battlePlays || scene.saleRuns) return undefined
    const last = Math.max(teased?.at ?? 0, ...ASKS.map((ask) => asked.shown[ask] ?? 0))
    if (now - last < ASK_SPACING_MS) return undefined
    let due: Ask | undefined
    for (const ask of ASKS) {
        if (!askedAgain(ask, asked.opened[ask], now)) continue
        if (due === undefined || (asked.shown[ask] ?? 0) < (asked.shown[due] ?? 0)) due = ask
    }
    return due
}

function askedAgain(ask: Ask, openedAt: number | undefined, now: number): boolean {
    if (openedAt === undefined) return true
    switch (ask) {
        case "coffee":
            return now - openedAt >= COFFEE_AGAIN_MS
        case "tiktok":
        case "youtube":
            return false
    }
}

export function askedOf(kept: string | null): Asked {
    if (kept === null) return NOTHING_ASKED
    try {
        const asked: unknown = JSON.parse(kept)
        if (typeof asked !== "object" || asked === null) return NOTHING_ASKED
        const {shown, opened} = asked as Record<string, unknown>
        return {shown: timesOf(shown), opened: timesOf(opened)}
    } catch {
        return NOTHING_ASKED
    }
}

function timesOf(kept: unknown): Partial<Record<Ask, number>> {
    if (typeof kept !== "object" || kept === null) return {}
    const times: Partial<Record<Ask, number>> = {}
    for (const ask of ASKS) {
        const at = (kept as Record<string, unknown>)[ask]
        if (typeof at === "number") times[ask] = at
    }
    return times
}

export function askClosed(asked: Asked, ask: Ask, now: number): Asked {
    return {...asked, shown: {...asked.shown, [ask]: now}}
}

export function askOpened(asked: Asked, ask: Ask, now: number): Asked {
    return {shown: {...asked.shown, [ask]: now}, opened: {...asked.opened, [ask]: now}}
}

export const SALE_SEEN_STORAGE_KEY = "clickplanet-sale-seen"

// A sale is shown on the first visit while it runs, once a day at most after that, and once more in its last hours.
export const SALE_NUDGE_EVERY_MS = DAY
export const SALE_LAST_CALL_MS = 6 * HOUR

// The sale this browser was shown last, by its key, and when.
export type SaleSeen = {
    key: string
    at: number
}

// A sale is known by its end.
export function saleKeyOf(sale: Sale): string {
    return String(sale.endsAt)
}

export function saleNudgeDue(sale: Sale | undefined, seen: SaleSeen | null, scene: NudgeScene, now: number): sale is Sale {
    if (!saleRuns(sale, now) || scene.guided || scene.battlePlays) return false
    if (seen?.key !== saleKeyOf(sale) || now - seen.at >= SALE_NUDGE_EVERY_MS) return true
    const lastCall = sale.endsAt - SALE_LAST_CALL_MS
    return now >= lastCall && seen.at < lastCall
}

export function saleSeenOf(kept: string | null): SaleSeen | null {
    if (kept === null) return null
    try {
        const seen: unknown = JSON.parse(kept)
        if (typeof seen !== "object" || seen === null) return null
        const {key, at} = seen as Record<string, unknown>
        return typeof key === "string" && typeof at === "number" ? {key, at} : null
    } catch {
        return null
    }
}
