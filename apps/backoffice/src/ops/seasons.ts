import {Duration} from "@bufbuild/protobuf"
import {GamemasterAdmin} from "../clients.ts"
import {localOf, timeOf} from "../components/time.ts"
import {GetSeasonsResponse, ScheduledSeason, SeasonFinale} from "../gen/grpc/gamemaster/v1/admin_pb.ts"
import {Cutoff, offCutoffWords} from "./cutoff.ts"

const HOUR = 3600
const DAY = 24 * HOUR

// A line of the seasons: a season, its times and its durations.
export type Row = {scheduled: ScheduledSeason, number: number, startsAt?: Date, endsAt?: Date}

// The seasons by number.
export function rowsOf(seasons: GetSeasonsResponse): Row[] {
    return seasons.seasons
        .map(scheduled => ({
            scheduled, number: scheduled.season?.number ?? 0, startsAt: scheduled.season?.startsAt?.toDate(),
            endsAt: scheduled.season?.endsAt?.toDate(),
        }))
        .sort((a, b) => a.number - b.number)
}

export type Phase = "done" | "running" | "to come"

export function phaseOf(row: Row, now: Date): Phase {
    if (row.endsAt !== undefined && row.endsAt <= now) return "done"
    if (row.startsAt !== undefined && row.startsAt <= now) return "running"
    return "to come"
}

// Only the last season, and only before it starts.
export function deletable(row: Row, rows: Row[], now: Date): boolean {
    return row === rows[rows.length - 1] && phaseOf(row, now) === "to come"
}

function secondsOf(duration: Duration | undefined): number {
    return duration === undefined ? 0 : Number(duration.seconds) + duration.nanos / 1e9
}

function durationOf(ms: number): Duration {
    const seconds = Math.trunc(ms / 1000)
    return new Duration({seconds: BigInt(seconds), nanos: (ms - seconds * 1000) * 1e6})
}

// "1 d 6 h", "28 d", "0 h".
export function spanOf(duration: Duration | undefined): string {
    const seconds = secondsOf(duration)
    const days = Math.floor(seconds / DAY)
    const hours = Math.round((seconds - days * DAY) / HOUR * 100) / 100
    if (days === 0) return `${hours} h`
    return hours === 0 ? `${days} d` : `${days} d ${hours} h`
}

// A season's last battle, as the form edits it: off for a season with none, its length in hours.
export type FinaleDraft = {on: boolean, hours: string, ruleSet: string, description: string, times: string}

// A season as the form edits it: its start and its end, each a datetime-local in the operator's own time.
export type Draft = {startsAt: string, endsAt: string, finale: FinaleDraft}

const NO_FINALE: FinaleDraft = {on: false, hours: "2", ruleSet: "", description: "", times: "3"}

function hoursOf(duration: Duration | undefined): string {
    return String(Math.round(secondsOf(duration) / HOUR * 1e6) / 1e6)
}

function finaleDraftOf(finale: SeasonFinale | undefined): FinaleDraft {
    if (finale === undefined) return NO_FINALE
    return {on: true, hours: hoursOf(finale.length), ruleSet: finale.ruleSet, description: finale.description, times: String(finale.times)}
}

// What a season's break counts from: the end of the season before; season 0's own start less its break.
function originOf(row: Row | undefined, rows: Row[]): Date | undefined {
    if (row === undefined) return rows[rows.length - 1]?.endsAt
    const before = rows.find(other => other.number === row.number - 1)
    if (before !== undefined) return before.endsAt
    return row.startsAt && new Date(row.startsAt.getTime() - secondsOf(row.scheduled.breakBefore) * 1000)
}

// A new season starts as the last one ends, and lasts 28 days.
export function draftOf(row: Row | undefined, rows: Row[]): Draft {
    if (row !== undefined) {
        return {startsAt: localOf(row.startsAt), endsAt: localOf(row.endsAt), finale: finaleDraftOf(row.scheduled.finale)}
    }
    const startsAt = originOf(undefined, rows)
    return {startsAt: localOf(startsAt), endsAt: localOf(startsAt && new Date(startsAt.getTime() + 28 * DAY * 1000)), finale: NO_FINALE}
}

const startKept = (draft: Draft, row: Row | undefined) => row !== undefined && draft.startsAt === localOf(row.startsAt)
const endKept = (draft: Draft, row: Row | undefined) => row !== undefined && draft.endsAt === localOf(row.endsAt)

// A time the form left as it was is the server's, to the millisecond.
function instantOf(local: string, kept: Date | undefined, isKept: boolean): Date | undefined {
    if (isKept) return kept
    const date = new Date(local)
    return local === "" || Number.isNaN(date.getTime()) ? undefined : date
}

// The times the form gives, and what the season's break counts from.
export type Window = {origin: Date, startsAt: Date, endsAt: Date}

export function windowOf(draft: Draft, row: Row | undefined, rows: Row[]): Window | undefined {
    const origin = originOf(row, rows)
    const startsAt = instantOf(draft.startsAt, row?.startsAt, startKept(draft, row))
    const endsAt = instantOf(draft.endsAt, row?.endsAt, endKept(draft, row))
    if (origin === undefined || startsAt === undefined || endsAt === undefined) return undefined
    return {origin, startsAt, endsAt}
}

// The break and the length the server keeps; one the form left as it was is sent as kept, to the nanosecond.
function breakOf(draft: Draft, row: Row | undefined, window: Window): Duration {
    if (startKept(draft, row) && row?.scheduled.breakBefore !== undefined) return row.scheduled.breakBefore
    return durationOf(window.startsAt.getTime() - window.origin.getTime())
}

function lengthOf(draft: Draft, row: Row | undefined, window: Window): Duration {
    if (startKept(draft, row) && endKept(draft, row) && row?.scheduled.length !== undefined) return row.scheduled.length
    return durationOf(window.endsAt.getTime() - window.startsAt.getTime())
}

// The season's length, and its break after the season before when there is one: "27 d 6 h long, starting 16 h after
// season 0 ends".
export function spansOf(draft: Draft, row: Row | undefined, rows: Row[]): string | undefined {
    const window = windowOf(draft, row, rows)
    if (window === undefined) return undefined
    const length = `${spanOf(lengthOf(draft, row, window))} long`
    const before = numberOf(row, rows) - 1
    if (before < 0) return length
    const brk = breakOf(draft, row, window)
    return `${length}, starting ${secondsOf(brk) === 0 ? "as" : `${spanOf(brk)} after`} season ${before} ends`
}

function numberOf(row: Row | undefined, rows: Row[]): number {
    return row?.number ?? (rows[rows.length - 1]?.number ?? -1) + 1
}

// When the final battle plays: its length before the season's end.
export function finaleWindowOf(draft: Draft, row: Row | undefined, rows: Row[]): {startsAt: Date, endsAt: Date} | undefined {
    const window = windowOf(draft, row, rows)
    const hours = Number(draft.finale.hours)
    if (!draft.finale.on || window === undefined || draft.finale.hours === "" || !Number.isFinite(hours) || hours <= 0) return undefined
    return {startsAt: new Date(window.endsAt.getTime() - hours * HOUR * 1000), endsAt: window.endsAt}
}

function finaleComplete(finale: FinaleDraft): boolean {
    if (!finale.on) return true
    const times = Number(finale.times)
    const hours = Number(finale.hours)
    return finale.hours !== "" && Number.isFinite(hours) && hours > 0 && finale.ruleSet.trim() !== "" &&
        Number.isInteger(times) && times >= 1 && times <= 10
}

export function complete(draft: Draft, row: Row | undefined, rows: Row[]): boolean {
    return windowOf(draft, row, rows) !== undefined && finaleComplete(draft.finale)
}

// What the server will refuse, as the form sees it. The server holds the rules: the form still sends.
export function flawsOf(draft: Draft, row: Row | undefined, rows: Row[], cutoff: Cutoff | undefined, now: Date): string[] {
    const window = windowOf(draft, row, rows)
    if (window === undefined) return []
    const flaws: string[] = []
    const before = numberOf(row, rows) - 1
    if (before >= 0 && window.startsAt < window.origin) {
        flaws.push(`It starts before season ${before} ends, at ${timeOf(window.origin)}.`)
    }
    if (!startKept(draft, row) && window.startsAt <= now) flaws.push("It starts before now.")
    if (window.endsAt <= window.startsAt) flaws.push("It ends before it starts.")
    if (!endKept(draft, row) && window.endsAt <= now) flaws.push("It ends before now.")
    const finale = finaleWindowOf(draft, row, rows)
    if (finale !== undefined && finale.startsAt < window.startsAt) flaws.push("The final battle is longer than the season.")
    if (finale !== undefined && cutoff !== undefined) {
        const off = offCutoffWords("The final battle", finale.startsAt, finale.endsAt, cutoff)
        if (off !== undefined) flaws.push(off)
    }
    return flaws
}

// "4 h, rush × 3", or none.
export function finaleSummaryOf(finale: SeasonFinale | undefined): string {
    return finale === undefined ? "none" : `${spanOf(finale.length)}, ${finale.ruleSet} × ${finale.times}`
}

// When the season's finale starts: its length before the season's end.
export function finaleStartOf(row: Row): Date | undefined {
    const finale = row.scheduled.finale
    if (finale === undefined || row.endsAt === undefined) return undefined
    return new Date(row.endsAt.getTime() - secondsOf(finale.length) * 1000)
}

// Once its finale started, a season's end and its finale stay.
export function finaleStarted(row: Row, now: Date): boolean {
    const startsAt = finaleStartOf(row)
    return startsAt !== undefined && startsAt <= now
}

// The finale the form sends: unset for none, and its length as kept when the form left it.
function finaleOf(finale: FinaleDraft, kept: SeasonFinale | undefined): SeasonFinale | undefined {
    if (!finale.on) return undefined
    const length = kept?.length !== undefined && finale.hours === hoursOf(kept.length)
        ? kept.length
        : new Duration({seconds: BigInt(Math.round(Number(finale.hours) * HOUR))})
    return new SeasonFinale({length, ruleSet: finale.ruleSet.trim(), description: finale.description, times: Number(finale.times)})
}

export function questionOf(draft: Draft, row: Row | undefined, rows: Row[]): string {
    const window = windowOf(draft, row, rows)
    const finale = finaleWindowOf(draft, row, rows)
    const dates = window === undefined ? "" : ` from ${timeOf(window.startsAt)} to ${timeOf(window.endsAt)} (${spansOf(draft, row, rows)})`
    const battle = finale === undefined
        ? ", with no final battle"
        : `, ending with a final battle of ${finaleSummaryOf(finaleOf(draft.finale, undefined))} from ${timeOf(finale.startsAt)}`
    return `${row === undefined ? "Add" : "Run"} season ${numberOf(row, rows)}${dates}${battle}?`
}

function valuesOf(draft: Draft, row: Row | undefined, rows: Row[]) {
    const window = windowOf(draft, row, rows)
    if (window === undefined) throw new Error("the season has no start or no end")
    return {
        breakBefore: breakOf(draft, row, window), length: lengthOf(draft, row, window),
        finale: finaleOf(draft.finale, row?.scheduled.finale),
    }
}

export async function addSeason(gamemaster: GamemasterAdmin, draft: Draft, rows: Row[]): Promise<number> {
    const added = await gamemaster.addSeason(valuesOf(draft, undefined, rows))
    return added.number
}

export async function editSeason(gamemaster: GamemasterAdmin, row: Row, draft: Draft, rows: Row[]): Promise<void> {
    await gamemaster.editSeason({number: row.number, ...valuesOf(draft, row, rows)})
}

export async function deleteSeason(gamemaster: GamemasterAdmin, number: number): Promise<void> {
    await gamemaster.deleteSeason({number})
}
