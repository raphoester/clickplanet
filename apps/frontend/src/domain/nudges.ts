import {Teased} from "./battleTeaser.ts"

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

export type AskScene = {
    guided: boolean
    battlePlays: boolean
    onCrazyGames: boolean
}

export function askDue(asked: Asked, teased: Teased | null, scene: AskScene, now: number): Ask | undefined {
    if (scene.guided || scene.battlePlays || scene.onCrazyGames) return undefined
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
