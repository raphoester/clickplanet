import {Season} from "../backends/season.ts"

const MINUTE = 60
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR

export type SeasonClock = {
    text: string
    finale: boolean
}

export function seasonClock(season: Season, now: number): SeasonClock | undefined {
    if (now >= season.endsAt) return undefined

    const seconds = Math.ceil((season.endsAt - now) / 1000)
    if (now >= season.finaleStartsAt) return {text: `Final Assault · ${countdown(seconds)}`, finale: true}
    if (seconds >= DAY) {
        return {text: `S${season.number} · ${Math.floor(seconds / DAY)}d ${twoDigits(seconds % DAY / HOUR)}h`, finale: false}
    }
    return {text: countdown(seconds), finale: false}
}

function countdown(seconds: number): string {
    if (seconds >= HOUR) return `${Math.floor(seconds / HOUR)}h ${twoDigits(seconds % HOUR / MINUTE)}m`
    return `${twoDigits(seconds / MINUTE)}:${twoDigits(seconds % MINUTE)}`
}

function twoDigits(n: number): string {
    return String(Math.floor(n)).padStart(2, "0")
}

export function seasonEnd(season: Season, timeZone?: string): string {
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

    return `Ends ${part("weekday")} ${part("day")} ${part("month")}, ${part("hour")}:${part("minute")}`
}
