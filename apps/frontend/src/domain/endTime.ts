const clock = new Intl.DateTimeFormat(undefined, {hour: "2-digit", minute: "2-digit"})
const calendar = new Intl.DateTimeFormat(undefined, {month: "short", day: "numeric", hour: "2-digit", minute: "2-digit"})
const yearly = new Intl.DateTimeFormat(undefined, {year: "numeric", month: "short", day: "numeric", hour: "2-digit", minute: "2-digit"})

const CLOCK_READS_FOR_MS = 20 * 3600_000

export function describeEnd(at: number, now: number): string {
    if (at - now < CLOCK_READS_FOR_MS) return clock.format(at)
    if (new Date(at).getFullYear() !== new Date(now).getFullYear()) return yearly.format(at)
    return calendar.format(at)
}
