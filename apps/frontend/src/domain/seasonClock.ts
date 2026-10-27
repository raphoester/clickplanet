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

export type SeasonCountdown = {
    label: string
    tag: string
    left: string
}

// The day's end while a day runs, the next battle's start, then the season's end. None once the season ended.
export function countdownsOf(season: Season, now: number, battle?: BattleToCome): SeasonCountdown[] {
    if (now >= season.endsAt) return []

    const ends = {label: "Season ends in", tag: "Season", left: timeLeft(countdownTo(season.endsAt, now))}
    const battles = battle && battle.startsAt > now
        ? [{label: "Next battle in", tag: "Battle", left: timeLeft(countdownTo(battle.startsAt, now))}]
        : []
    const cutoff = cutoffOf(season, now)
    if (now >= season.lastDayEndsAt || cutoff === season.endsAt) return [...battles, ends]
    return [{label: "Today ends in", tag: "Today", left: timeLeft(countdownTo(cutoff, now))}, ...battles, ends]
}

export type EndClock = {
    left: string
    countdown: Countdown
}

// The time left in the season. None once it ended.
export function endClock(season: Season, now: number): EndClock | undefined {
    if (now >= season.endsAt) return undefined

    const countdown = countdownTo(season.endsAt, now)
    return {left: timeLeft(countdown), countdown}
}

// Every day ends at the time of day of the season's last cutoff.
function cutoffOf(season: Season, now: number): number {
    const daysAfter = Math.floor((season.lastDayEndsAt - now - 1) / (DAY * 1000))
    return season.lastDayEndsAt - daysAfter * DAY * 1000
}

function countdownTo(at: number, now: number): Countdown {
    const seconds = Math.ceil((at - now) / 1000)
    return {
        days: Math.floor(seconds / DAY),
        hours: Math.floor(seconds % DAY / HOUR),
        minutes: Math.floor(seconds % HOUR / MINUTE),
        seconds: seconds % MINUTE,
    }
}

function timeLeft({days, hours, minutes, seconds}: Countdown): string {
    const minutesAndSeconds = `${twoDigits(minutes)}m ${twoDigits(seconds)}s`
    if (days > 0) return `${days}d ${twoDigits(hours)}h ${minutesAndSeconds}`
    if (hours > 0) return `${hours}h ${minutesAndSeconds}`
    return minutesAndSeconds
}

export function timeLeftTo(at: number, now: number): string {
    return timeLeft(countdownTo(Math.max(at, now), now))
}

export function shortLeft(left: string): string {
    return left.split(" ").slice(0, 2).join(" ")
}

export function twoDigits(n: number): string {
    return String(n).padStart(2, "0")
}

// "Sat 31 Oct · 23:00", in the player's time.
export function endsLine(season: Season, timeZone?: string): string {
    const parts = new Intl.DateTimeFormat("en-GB", {
        weekday: "short",
        day: "numeric",
        month: "short",
        hour: "2-digit",
        minute: "2-digit",
        hourCycle: "h23",
        timeZone,
    }).formatToParts(season.endsAt)
    const part = (type: Intl.DateTimeFormatPartTypes) => parts.find((p) => p.type === type)?.value ?? ""

    return `${part("weekday")} ${part("day")} ${part("month")} · ${part("hour")}:${part("minute")}`
}
