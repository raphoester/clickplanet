const FORMAT = new Intl.DateTimeFormat(undefined, {dateStyle: "medium", timeStyle: "short"})

export function timeOf(date: Date): string {
    return FORMAT.format(date)
}

const CLOCK = new Intl.DateTimeFormat(undefined, {timeStyle: "short"})

// The time of day alone, for a line whose day is shown elsewhere.
export function clockOf(date: Date): string {
    return CLOCK.format(date)
}
