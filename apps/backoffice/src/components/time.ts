const FORMAT = new Intl.DateTimeFormat(undefined, {dateStyle: "medium", timeStyle: "short"})

export function timeOf(date: Date): string {
    return FORMAT.format(date)
}
