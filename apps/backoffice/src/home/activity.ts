import {Duration, Timestamp} from "@bufbuild/protobuf"
import {CountryActivity, GetActivityResponse} from "../gen/grpc/planet/v1/admin_pb.ts"

export type Span = {hours: number, label: string}

export const SPANS: Span[] = [
    {hours: 6, label: "6 hours"},
    {hours: 24, label: "24 hours"},
    {hours: 72, label: "3 days"},
]

export const FIRST_SPAN = SPANS[1]

export function durationOf(span: Span): Duration {
    return new Duration({seconds: BigInt(span.hours * 3600)})
}

export type Slot = {start: Date, end: Date, clicks: number, tiles: number, players: number, filling: boolean}

export function slotsOf(activity: GetActivityResponse): Slot[] {
    const since = dateOf(activity.since)
    const until = dateOf(activity.until)
    const step = millisOf(activity.step)

    return activity.slots.map((slot, i) => {
        const start = new Date(since.getTime() + i * step)
        const end = new Date(start.getTime() + step)
        return {start, end, clicks: slot.clicks, tiles: slot.tiles, players: slot.players, filling: end > until}
    })
}

export function stepOf(activity: GetActivityResponse): number {
    return millisOf(activity.step)
}

function dateOf(at: Timestamp | undefined): Date {
    return at === undefined ? new Date(0) : at.toDate()
}

function millisOf(duration: Duration | undefined): number {
    if (duration === undefined) return 0
    return Number(duration.seconds) * 1000 + duration.nanos / 1e6
}

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

export type Order = "held" | "won" | "lost"

export function countriesBy(countries: CountryActivity[], order: Order): CountryActivity[] {
    if (order === "held") return countries
    return [...countries].sort((a, b) => b[order] - a[order] || b.held - a.held)
}

export function heldOf(countries: CountryActivity[]): number {
    return countries.reduce((sum, country) => sum + country.held, 0)
}
