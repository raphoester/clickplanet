import {Season} from "../backends/season.ts"
import {BattleToCome} from "../backends/turn.ts"

const MINUTE = 60
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR

export type Countdown = {
    days: number
    hours: number
    minutes: number
    seconds: number
}

export type CountdownKind = "today" | "battle" | "final" | "season"

export type SeasonCountdown = {
    kind: CountdownKind
    countdown: Countdown
}

// The day's end while a day runs, the next battle's start, then the season's end. None once the season ended.
export function countdownsOf(season: Season, now: number, battle?: BattleToCome, dayEndsAt?: number): SeasonCountdown[] {
    if (now >= season.endsAt) return []

    const ends: SeasonCountdown = {kind: "season", countdown: countdownTo(season.endsAt, now)}
    const battles: SeasonCountdown[] = battle && battle.startsAt > now
        ? [{kind: battle.endsSeason ? "final" : "battle", countdown: countdownTo(battle.startsAt, now)}]
        : []
    if (dayEndsAt === undefined || dayEndsAt <= now || dayEndsAt >= season.endsAt) return [...battles, ends]
    return [{kind: "today", countdown: countdownTo(dayEndsAt, now)}, ...battles, ends]
}

// The time left in the season. None once it ended.
export function endClock(season: Season, now: number): Countdown | undefined {
    if (now >= season.endsAt) return undefined
    return countdownTo(season.endsAt, now)
}

export function countdownTo(at: number, now: number): Countdown {
    const seconds = Math.ceil((at - now) / 1000)
    return {
        days: Math.floor(seconds / DAY),
        hours: Math.floor(seconds % DAY / HOUR),
        minutes: Math.floor(seconds % HOUR / MINUTE),
        seconds: seconds % MINUTE,
    }
}

export function timeLeftTo(at: number, now: number): Countdown {
    return countdownTo(Math.max(at, now), now)
}
