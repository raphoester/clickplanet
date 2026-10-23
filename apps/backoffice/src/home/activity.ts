export type Span = {hours: number, label: string, step: number}

export const SPANS: Span[] = [
    {hours: 6, label: "6 hours", step: 5 * 60_000},
    {hours: 24, label: "24 hours", step: 15 * 60_000},
    {hours: 72, label: "3 days", step: 3600_000},
]

export const FIRST_SPAN = SPANS[1]

export type Slot = {start: Date, end: Date, clicks: number, tiles: number, players: number, filling: boolean}

export function ticksOf(highest: number): number[] {
    if (highest <= 0) return [0, 1]

    const rough = highest / 3
    const magnitude = 10 ** Math.floor(Math.log10(rough))
    // Every chart counts: no step under one.
    const step = Math.max(1, [1, 2, 5, 10].map(m => m * magnitude).find(candidate => candidate >= rough) ?? 10 * magnitude)
    const top = Math.ceil(highest / step) * step

    const ticks: number[] = []
    for (let tick = 0; tick <= top + step / 2; tick += step) ticks.push(tick)
    return ticks
}

const COMPACT = new Intl.NumberFormat("en", {notation: "compact", maximumFractionDigits: 1})
const WHOLE = new Intl.NumberFormat("en")

export function compactOf(n: number): string {
    return COMPACT.format(n)
}

export function wholeOf(n: number): string {
    return WHOLE.format(n)
}

// About six time labels across any span.
export function labelEveryOf(step: number): number {
    if (step <= 5 * 60_000) return 1
    if (step <= 15 * 60_000) return 4
    return 12
}

export function labelled(slot: Slot, every: number): boolean {
    return slot.start.getMinutes() === 0 && slot.start.getHours() % every === 0
}

const HOUR = new Intl.DateTimeFormat(undefined, {hour: "2-digit", minute: "2-digit"})
const DAY = new Intl.DateTimeFormat(undefined, {weekday: "short", day: "numeric"})
const SLOT = new Intl.DateTimeFormat(undefined, {weekday: "short", hour: "2-digit", minute: "2-digit"})

export function tickLabelOf(slot: Slot, every: number): string {
    return every === 12 && slot.start.getHours() === 0 ? DAY.format(slot.start) : HOUR.format(slot.start)
}

export function slotLabelOf(slot: Slot): string {
    return `${SLOT.format(slot.start)} – ${HOUR.format(slot.end)}`
}

export type Totals = {clicks: number, tiles: number}

export function totalsOf(slots: Slot[]): Totals {
    return slots.reduce((sum, slot) => ({clicks: sum.clicks + slot.clicks, tiles: sum.tiles + slot.tiles}), {clicks: 0, tiles: 0})
}
