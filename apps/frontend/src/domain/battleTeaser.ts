import {BattleToCome} from "../backends/turn.ts"
import {finaleLengthLine, partsOf} from "./battle.ts"

const HOUR = 60 * 60_000

export const BATTLE_TEASED_STORAGE_KEY = "clickplanet-battle-teased"

// A battle is teased in the day before it starts, as a page opens, and again once a few hours passed: a player who
// comes back in the evening hears of it again, one who reloads does not.
export const BATTLE_TEASER_WITHIN_MS = 24 * HOUR
export const BATTLE_TEASER_EVERY_MS = 6 * HOUR

// The battle this browser was told of last, and when.
export type Teased = {
    entryId: string
    at: number
}

export function teasedOf(kept: string | null): Teased | null {
    if (kept === null) return null
    try {
        const teased: unknown = JSON.parse(kept)
        if (typeof teased !== "object" || teased === null) return null
        const {entryId, at} = teased as Record<string, unknown>
        return typeof entryId === "string" && typeof at === "number" ? {entryId, at} : null
    } catch {
        return null
    }
}

// The next battle, while it starts within a day and this browser was not told of it in the last hours. The season's
// finale has its own teaser. `ready` holds a newcomer back until it played (see useBattleTeaser).
export function battleTeaserDue(battle: BattleToCome | undefined, teased: Teased | null, now: number, ready: boolean): battle is BattleToCome {
    if (!battle || battle.endsSeason || battle.entryId === "" || !ready) return false
    if (battle.startsAt <= now || battle.startsAt - now > BATTLE_TEASER_WITHIN_MS) return false
    return teased?.entryId !== battle.entryId || now - teased.at >= BATTLE_TEASER_EVERY_MS
}

// "Today · 22:00 – 23:00", "Tomorrow · 22:00 – 23:00" or "Sat 7 Nov · 22:00 – 23:00", in the player's time.
export function battleWhenLine(battle: BattleToCome, now: number, timeZone?: string): string {
    return `${dayWordOf(battle.startsAt, now, timeZone)} · ${clockOf(battle.startsAt, timeZone)} – ${clockOf(battle.endsAt, timeZone)}`
}

export type TeaserCardKind = "war" | "points" | "rally"

export type TeaserCard = {
    kind: TeaserCardKind
    head: string
    text: string
}

// The three cards' words: what the battle is, what it is worth, and when to be there. Only the schedule's facts and
// what is always true: what the rule set changes is the operator's description, above them.
export function teaserCardsOf(battle: BattleToCome, timeZone?: string): TeaserCard[] {
    const ends = clockOf(battle.endsAt, timeZone)
    return [
        {
            kind: "war",
            head: `${finaleLengthLine(battle)} of war`,
            text: "Every country fights at once. Keep your bombs and shields for it.",
        },
        battle.times > 1
            ? {kind: "points", head: `Points ×${battle.times}`, text: `The countries holding the most ground at ${ends} score the points of ${battle.times} days.`}
            : {kind: "points", head: "A round of its own", text: `The countries holding the most ground at ${ends} score its points.`},
        {
            kind: "rally",
            head: `Be there at ${clockOf(battle.startsAt, timeZone)}`,
            text: "Bring your friends. Your flag needs every click.",
        },
    ]
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
