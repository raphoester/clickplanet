import {BattleToCome, Turn} from "../backends/turn.ts"
import {timeLeftTo} from "./seasonClock.ts"

export const BATTLE_SEEN_STORAGE_KEY = "clickplanet-battle-seen"
export const BATTLE_OVER_MS = 5000

export type Battle = {
    entryId: string
    since: number
    endsAt: number
    description: string
    // Its round in scoring; empty for a battle that scores nothing.
    roundId: string
    // The season's finale: its last battle.
    endsSeason?: boolean
}

export function battleOf(turn: Turn | undefined): Battle | undefined {
    if (!turn || turn.kind !== "battle" || turn.endsAt === undefined) return undefined
    return {
        entryId: turn.entryId, since: turn.since ?? 0, endsAt: turn.endsAt, description: turn.description, roundId: turn.roundId,
        endsSeason: turn.endsSeason,
    }
}

// What the players call a battle: the season's finale is its final battle.
export function battleNameOf(battle: {endsSeason?: boolean}): string {
    return battle.endsSeason ? "Final Battle" : "Battle"
}

// The server's list is of its read: a battle whose start passed since is no longer to come.
export function nextBattleOf(battles: BattleToCome[], now: number): BattleToCome | undefined {
    return battles.find((battle) => battle.startsAt > now)
}

// A season's finale has no entry: its round names it.
export function battleKeyOf(battle: Battle): string {
    return battle.entryId || battle.roundId
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
