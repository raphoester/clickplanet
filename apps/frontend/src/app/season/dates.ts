import {Season} from "../../backends/season.ts"
import {translator} from "../../i18n/page.ts"

const DAY_AND_TIME: Intl.DateTimeFormatOptions = {weekday: "short", day: "numeric", month: "short", hour: "2-digit", minute: "2-digit"}
const TIME: Intl.DateTimeFormatOptions = {hour: "2-digit", minute: "2-digit"}

export function dayAndTime(at: number): string {
    return translator().dateTime(at, DAY_AND_TIME)
}

export function dayAndTimeRange(from: number, to: number): string {
    return translator().dateTimeRange(from, to, DAY_AND_TIME)
}

export function timeOfDay(at: number): string {
    return translator().dateTime(at, TIME)
}

export function timeRange(from: number, to: number): string {
    return translator().dateTimeRange(from, to, TIME)
}

export function seasonSpan(season: Season): string {
    return translator().dateTimeRange(season.startsAt, season.endsAt, {day: "numeric", month: "short", year: "numeric"})
}
