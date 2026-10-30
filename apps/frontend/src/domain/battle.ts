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

export const FINALE_TEASED_STORAGE_KEY = "clickplanet-finale-teased"
export const FINALE_NEAR_MS = 48 * 60 * 60_000

export type Finale = {
    roundId: string
    startsAt: number
    endsAt: number
    times: number
    description: string
    live: boolean
}

// The schedule lists the season's finale until it starts, and the turn plays it until it ends.
export function finaleOf(battles: BattleToCome[], turn: Turn | undefined, now: number): Finale | undefined {
    if (turn?.kind === "battle" && turn.endsSeason && turn.endsAt !== undefined && turn.endsAt > now) {
        const {roundId, endsAt, times, description} = turn
        return {roundId, startsAt: turn.since ?? now, endsAt, times, description, live: true}
    }
    const listed = battles.find((battle) => battle.endsSeason && battle.endsAt > now)
    if (!listed) return undefined
    const {roundId, startsAt, endsAt, times, description} = listed
    return {roundId, startsAt, endsAt, times, description, live: startsAt <= now}
}

export function finaleNear(finale: Finale | undefined, now: number): finale is Finale {
    return finale !== undefined && !finale.live && finale.startsAt - now <= FINALE_NEAR_MS
}

// A newcomer plays before a full screen talks of a battle: half the start bank, before it runs dry.
export const FINALE_TEASER_AFTER_CLICKS = 30

// Once per browser and per finale, in the days before it starts, once the page took enough clicks.
export function finaleTeaserDue(finale: Finale | undefined, seen: string | null, now: number, clicks: number): finale is Finale {
    return finaleNear(finale, now) && finale.roundId !== seen && clicks >= FINALE_TEASER_AFTER_CLICKS
}

// "Sat 31 Oct · 21:00 – 23:00", in the player's time.
export function finaleWhenLine(finale: Finale, timeZone?: string): string {
    const start = partsOf(finale.startsAt, timeZone)
    const end = partsOf(finale.endsAt, timeZone)
    return `${start("weekday")} ${start("day")} ${start("month")} · ${start("hour")}:${start("minute")} – ${end("hour")}:${end("minute")}`
}

// "2 hours", "1 hour", or "90 minutes" when it is not whole hours.
export function finaleLengthLine(finale: Finale): string {
    const minutes = Math.round((finale.endsAt - finale.startsAt) / 60_000)
    if (minutes % 60 !== 0) return `${minutes} minutes`
    return minutes === 60 ? "1 hour" : `${minutes / 60} hours`
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
