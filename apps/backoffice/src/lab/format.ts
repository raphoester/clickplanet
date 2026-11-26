import {compactOf} from "../home/activity.ts"

const DAY = new Intl.DateTimeFormat("en-GB", {day: "numeric", month: "short", timeZone: "UTC"})
const TIME = new Intl.DateTimeFormat("en-GB", {hour: "2-digit", minute: "2-digit", hourCycle: "h23", timeZone: "UTC"})

export function dayLabelOf(day: string): string {
    return DAY.format(new Date(`${day}T00:00:00Z`))
}

// 20 Nov, 20–23 Nov, 30 Nov – 2 Dec.
export function daysOf(first: string, last: string): string {
    const from = dayLabelOf(first)
    if (first === last) return from
    const to = dayLabelOf(last)
    return first.slice(0, 7) === last.slice(0, 7) ? `${from.split(" ")[0]}–${to}` : `${from} – ${to}`
}

export function momentOf(at: Date): string {
    return `${DAY.format(at)} ${TIME.format(at)}`
}

export function count(value: number | undefined): string {
    return value === undefined ? "—" : Math.round(value).toLocaleString("en-US")
}

// In full under 10,000, compact past it, as an axis reads.
export function viewsOf(value: number | undefined): string {
    return value === undefined || value < 10_000 ? count(value) : compactOf(value)
}

export function percent(value: number | undefined): string {
    return value === undefined ? "—" : `${Math.round(value * 100)}%`
}

export function secondsOf(value: number | undefined): string {
    return value === undefined ? "—" : `${value.toFixed(1)} s`
}

export function rate(value: number | undefined): string {
    return value === undefined ? "—" : value.toFixed(1)
}
