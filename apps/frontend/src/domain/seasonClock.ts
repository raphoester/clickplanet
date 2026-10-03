import {Season} from "../backends/season.ts"

const MINUTE = 60
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR

export type SeasonClock = {
    finale: boolean
    left: string
    line: string
}

export function seasonClock(season: Season, now: number): SeasonClock | undefined {
    if (now >= season.endsAt) return undefined

    const finale = now >= season.finaleStartsAt
    const left = timeLeft(Math.ceil((season.endsAt - now) / 1000))
    return {finale, left, line: `${finale ? "Final Assault" : `Season ${season.number}`} · ${left}`}
}

function timeLeft(seconds: number): string {
    if (seconds >= DAY) return `${Math.floor(seconds / DAY)}d ${twoDigits(seconds % DAY / HOUR)}h`
    if (seconds >= HOUR) return `${Math.floor(seconds / HOUR)}h ${twoDigits(seconds % HOUR / MINUTE)}m`
    return `${twoDigits(seconds / MINUTE)}:${twoDigits(seconds % MINUTE)}`
}

function twoDigits(n: number): string {
    return String(Math.floor(n)).padStart(2, "0")
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
