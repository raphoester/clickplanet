const CLOCK_READS_FOR_MS = 20 * 3600_000

const CLOCK: Intl.DateTimeFormatOptions = {hour: "2-digit", minute: "2-digit"}
const CALENDAR: Intl.DateTimeFormatOptions = {month: "short", day: "numeric", hour: "2-digit", minute: "2-digit"}
const YEARLY: Intl.DateTimeFormatOptions = {year: "numeric", month: "short", day: "numeric", hour: "2-digit", minute: "2-digit"}

// The parts of an end worth saying: the time alone when it is near, the date too beyond, the year in another year.
export function endPartsOf(at: number, now: number): Intl.DateTimeFormatOptions {
    if (at - now < CLOCK_READS_FOR_MS) return CLOCK
    if (new Date(at).getFullYear() !== new Date(now).getFullYear()) return YEARLY
    return CALENDAR
}
