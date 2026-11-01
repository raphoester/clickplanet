const FORMAT = new Intl.DateTimeFormat(undefined, {dateStyle: "medium", timeStyle: "short"})

export function timeOf(date: Date): string {
    return FORMAT.format(date)
}

const CLOCK = new Intl.DateTimeFormat(undefined, {timeStyle: "short"})

// The time of day alone, for a line whose day is shown elsewhere.
export function clockOf(date: Date): string {
    return CLOCK.format(date)
}

// A time as a datetime-local field holds it, in the operator's own time; "" for none.
export function localOf(date: Date | undefined): string {
    if (date === undefined) return ""
    const pad = (n: number) => String(n).padStart(2, "0")
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

// The operator's own time zone, which every time on the page is in: "Europe/Sofia".
export function zoneOf(): string {
    return Intl.DateTimeFormat().resolvedOptions().timeZone
}

// How long ago, as a list reads it: "just now", "12 min ago", "3 h ago", "2 d ago".
export function agoOf(date: Date, now: Date = new Date()): string {
    const minutes = Math.floor((now.getTime() - date.getTime()) / 60_000)
    if (minutes < 1) return "just now"
    if (minutes < 60) return `${minutes} min ago`
    const hours = Math.floor(minutes / 60)
    if (hours < 48) return `${hours} h ago`
    return `${Math.floor(hours / 24)} d ago`
}

// A span of seconds in the unit it reads best in: "0.3 s", "5 min", "2 h", "1095 d".
export function secondsWord(seconds: number): string {
    if (!Number.isFinite(seconds)) return ""
    for (const [unit, size] of [["d", 86_400], ["h", 3600], ["min", 60]] as const) {
        if (seconds >= size && seconds % size === 0) return `${seconds / size} ${unit}`
    }
    return `${seconds} s`
}
