import {BattleToCome, Turn} from "../backends/turn.ts"
import {battleOf, partsOf} from "./battle.ts"
import {timeLeftTo} from "./seasonClock.ts"

const HOUR = 60 * 60_000

export const BATTLE_TEASED_STORAGE_KEY = "clickplanet-battle-teased"

// A battle is teased in the day before it starts, as a page opens, and again once a few hours passed: a player who
// comes back in the evening hears of it again, one who reloads does not.
const BATTLE_TEASER_WITHIN_MS = 24 * HOUR
export const BATTLE_TEASER_EVERY_MS = 6 * HOUR

// A newcomer plays before a full screen talks of a battle: half the start bank, before it runs dry.
export const TEASER_AFTER_CLICKS = 30

// The battle this browser was told of last, and when.
export type Teased = {
    key: string
    at: number
}

// A battle's key: its entry's id, or a season's finale's round, which has no entry.
export function teaserKeyOf(battle: BattleToCome): string {
    return battle.entryId || battle.roundId
}

export function teasedOf(kept: string | null): Teased | null {
    if (kept === null) return null
    try {
        const teased: unknown = JSON.parse(kept)
        if (typeof teased !== "object" || teased === null) return null
        const {key, at} = teased as Record<string, unknown>
        return typeof key === "string" && typeof at === "number" ? {key, at} : null
    } catch {
        return null
    }
}

// The next battle, Final Battle included, while it starts within a day and this browser was not told of it in the last
// hours. Whether it has cards is promo's to say: a battle with none is not teased.
export function battleTeaserDue(battle: BattleToCome | undefined, teased: Teased | null, now: number, ready: boolean): battle is BattleToCome {
    if (!battle || teaserKeyOf(battle) === "" || !ready) return false
    if (battle.startsAt <= now || battle.startsAt - now > BATTLE_TEASER_WITHIN_MS) return false
    return teased?.key !== teaserKeyOf(battle) || now - teased.at >= BATTLE_TEASER_EVERY_MS
}

// The battle that plays, as its teaser shows it: a player asks for its rules while it plays too.
export function playingOf(turn: Turn | undefined): BattleToCome | undefined {
    const battle = battleOf(turn)
    if (!turn || !battle) return undefined
    const {entryId, roundId, description, times, endsSeason} = turn
    return {entryId, roundId, description, times, endsSeason, startsAt: battle.since, endsAt: battle.endsAt}
}

// "Starts in 2h 05m 12s" before the battle, "Ends in 42m 10s" while it plays.
export function teaserClockLine(battle: BattleToCome, now: number): string {
    return battle.startsAt > now
        ? `Starts in ${timeLeftTo(battle.startsAt, now)}`
        : `Ends in ${timeLeftTo(battle.endsAt, now)}`
}

// "Today · 22:00 – 23:00", "Tomorrow · 22:00 – 23:00" or "Sat 7 Nov · 22:00 – 23:00", in the player's time.
export function battleWhenLine(battle: BattleToCome, now: number, timeZone?: string): string {
    return `${dayWordOf(battle.startsAt, now, timeZone)} · ${clockOf(battle.startsAt, timeZone)} – ${clockOf(battle.endsAt, timeZone)}`
}

function clockOf(ms: number, timeZone?: string): string {
    const part = partsOf(ms, timeZone)
    return `${part("hour")}:${part("minute")}`
}

function dayWordOf(ms: number, now: number, timeZone?: string): string {
    const day = partsOf(ms, timeZone)
    const sameDay = (other: number) => {
        const it = partsOf(other, timeZone)
        return it("day") === day("day") && it("month") === day("month")
    }
    if (sameDay(now)) return "Today"
    if (sameDay(now + 24 * HOUR)) return "Tomorrow"
    return `${day("weekday")} ${day("day")} ${day("month")}`
}
