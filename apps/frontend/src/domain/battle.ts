import {Turn} from "../backends/turn.ts"
import {timeLeftTo} from "./seasonClock.ts"

export const BATTLE_SEEN_STORAGE_KEY = "clickplanet-battle-seen"
export const TURN_REREAD_MS = 60_000
export const BATTLE_OVER_MS = 5000

export type Battle = {
    since: number
    endsAt: number
    description: string
}

export function battleOf(turn: Turn | undefined): Battle | undefined {
    if (!turn || turn.kind !== "battle" || turn.endsAt === undefined) return undefined
    return {since: turn.since ?? 0, endsAt: turn.endsAt, description: turn.description}
}

export function nextReadAt(turn: Turn | undefined, now: number): number {
    const soon = now + TURN_REREAD_MS
    if (turn?.endsAt !== undefined && turn.endsAt > now) return Math.min(soon, turn.endsAt)
    return soon
}

export function battleKeyOf(battle: Battle): string {
    return `${battle.since}:${battle.endsAt}`
}

export function battleDue(battle: Battle | undefined, seen: string | null): battle is Battle {
    return battle !== undefined && battleKeyOf(battle) !== seen
}

export function battleLeft(battle: Battle, now: number): string {
    return timeLeftTo(battle.endsAt, now)
}

export function battleEndsLine(battle: Battle, now: number, timeZone?: string): string {
    const end = partsOf(battle.endsAt, timeZone)
    const today = partsOf(now, timeZone)
    const time = `${end("hour")}:${end("minute")}`
    const sameDay = end("day") === today("day") && end("month") === today("month")
    return sameDay ? `Ends at ${time}` : `Ends ${end("weekday")} at ${time}`
}

function partsOf(ms: number, timeZone?: string): (type: Intl.DateTimeFormatPartTypes) => string {
    const parts = new Intl.DateTimeFormat("en-GB", {
        weekday: "short",
        day: "numeric",
        month: "short",
        hour: "2-digit",
        minute: "2-digit",
        hourCycle: "h23",
        timeZone,
    }).formatToParts(ms)

    return (type) => parts.find((p) => p.type === type)?.value ?? ""
}
