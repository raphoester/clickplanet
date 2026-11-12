import {BattleToCome, Turn} from "../backends/turn.ts"
import {Countdown, timeLeftTo} from "./seasonClock.ts"

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

// The server's list is of its read: a battle whose start passed since is no longer to come.
export function nextBattleOf(battles: BattleToCome[], now: number): BattleToCome | undefined {
    return battles.find((battle) => battle.startsAt > now)
}

const FINALE_NEAR_MS = 48 * 60 * 60_000

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

export function finaleMinutes(finale: Finale): number {
    return Math.round((finale.endsAt - finale.startsAt) / 60_000)
}

// A season's finale has no entry: its round names it.
export function battleKeyOf(battle: Battle): string {
    return battle.entryId || battle.roundId
}

export function battleDue(battle: Battle | undefined, seen: string | null): battle is Battle {
    return battle !== undefined && battleKeyOf(battle) !== seen
}

export function battleLeft(battle: Battle, now: number): Countdown {
    return timeLeftTo(battle.endsAt, now)
}

export function endsToday(battle: Battle, now: number): boolean {
    return sameDay(battle.endsAt, now)
}

// In the player's time zone.
export function sameDay(a: number, b: number): boolean {
    const x = new Date(a)
    const y = new Date(b)
    return x.getFullYear() === y.getFullYear() && x.getMonth() === y.getMonth() && x.getDate() === y.getDate()
}
