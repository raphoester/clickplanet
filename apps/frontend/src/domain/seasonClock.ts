import {Season} from "../backends/season.ts"

const MINUTE = 60
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR

export type SeasonClock = {
    finale: boolean
    left: string
}

export type Countdown = {
    days: number
    hours: number
    minutes: number
    seconds: number
}

export type FinaleClock = {
    live: boolean
    left: string
    countdown: Countdown
}

export function seasonClock(season: Season, now: number): SeasonClock | undefined {
    if (now >= season.endsAt) return undefined
    if (now >= season.finaleStartsAt) return {finale: true, left: timeLeft(countdownTo(season.endsAt, now))}

    const daysAfter = Math.floor((season.finaleStartsAt - now - 1) / (DAY * 1000))
    return {finale: false, left: timeLeft(countdownTo(season.finaleStartsAt - daysAfter * DAY * 1000, now))}
}

export function finaleClock(season: Season, now: number): FinaleClock | undefined {
    if (now >= season.endsAt) return undefined

    const live = now >= season.finaleStartsAt
    const countdown = countdownTo(live ? season.endsAt : season.finaleStartsAt, now)
    return {live, left: timeLeft(countdown), countdown}
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

export function twoDigits(n: number): string {
    return String(n).padStart(2, "0")
}

export type FinaleWindow = {
    day: string
    from: string
    to: string
}

export function finaleWindow(season: Season, timeZone?: string): FinaleWindow {
    const start = partsOf(season.finaleStartsAt, timeZone)
    const end = partsOf(season.endsAt, timeZone)

    return {
        day: `${start("weekday")} ${start("day")} ${start("month")}`,
        from: `${start("hour")}:${start("minute")}`,
        to: `${end("hour")}:${end("minute")}`,
    }
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
