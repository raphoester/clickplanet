const count = new Intl.NumberFormat()

export function days(n: number): string {
    return n === 1 ? "1 day" : `${count.format(n)} days`
}
