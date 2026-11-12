import {BattleToCome, Turn} from "../backends/turn.ts"
import {battleOf, sameDay} from "./battle.ts"
import {Countdown, timeLeftTo} from "./seasonClock.ts"

const HOUR = 60 * 60_000

export const BATTLE_TEASED_STORAGE_KEY = "clickplanet-battle-teased"

// A battle is teased in the day before it starts, again once a few hours passed, and once more in its last hour: a
// player who comes back in the evening hears of it again, one who reloads does not.
const BATTLE_TEASER_WITHIN_MS = 24 * HOUR
export const BATTLE_TEASER_EVERY_MS = 3 * HOUR
export const BATTLE_TEASER_LAST_CALL_MS = HOUR

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
// hours or in its last hour. Whether it has cards is promo's to say: a battle with none is not teased.
export function battleTeaserDue(battle: BattleToCome | undefined, teased: Teased | null, now: number, ready: boolean): battle is BattleToCome {
    if (!battle || teaserKeyOf(battle) === "" || !ready) return false
    if (battle.startsAt <= now || battle.startsAt - now > BATTLE_TEASER_WITHIN_MS) return false
    if (teased?.key !== teaserKeyOf(battle) || now - teased.at >= BATTLE_TEASER_EVERY_MS) return true
    const lastCall = battle.startsAt - BATTLE_TEASER_LAST_CALL_MS
    return now >= lastCall && teased.at < lastCall
}

// The battle that plays, as its teaser shows it: a player asks for its rules while it plays too.
export function playingOf(turn: Turn | undefined): BattleToCome | undefined {
    const battle = battleOf(turn)
    if (!turn || !battle) return undefined
    const {entryId, roundId, description, times, endsSeason} = turn
    return {entryId, roundId, description, times, endsSeason, startsAt: battle.since, endsAt: battle.endsAt}
}

export type TeaserClock = {
    to: "start" | "end"
    countdown: Countdown
}

export function teaserClockOf(battle: BattleToCome, now: number): TeaserClock {
    return battle.startsAt > now
        ? {to: "start", countdown: timeLeftTo(battle.startsAt, now)}
        : {to: "end", countdown: timeLeftTo(battle.endsAt, now)}
}

export type BattleDay = "today" | "tomorrow" | "later"

export function battleDayOf(battle: BattleToCome, now: number): BattleDay {
    if (sameDay(battle.startsAt, now)) return "today"
    if (sameDay(battle.startsAt, now + 24 * HOUR)) return "tomorrow"
    return "later"
}
