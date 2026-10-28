import {Duration} from "@bufbuild/protobuf"
import {GamemasterAdmin} from "../clients.ts"
import {GetSeasonsResponse, ScheduledSeason} from "../gen/grpc/gamemaster/v1/admin_pb.ts"

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

// "1 d 6 h", "28 d", "0 h".
export function spanOf(duration: Duration | undefined): string {
    const seconds = secondsOf(duration)
    const days = Math.floor(seconds / DAY)
    const hours = Math.round((seconds - days * DAY) / HOUR * 100) / 100
    if (days === 0) return `${hours} h`
    return hours === 0 ? `${days} d` : `${days} d ${hours} h`
}

// A duration as the form edits it: whole days, and hours.
export type Span = {days: string, hours: string}

function spanDraftOf(duration: Duration | undefined): Span {
    const seconds = secondsOf(duration)
    const days = Math.floor(seconds / DAY)
    return {days: String(days), hours: String(Math.round((seconds - days * DAY) / HOUR * 1e6) / 1e6)}
}

function spanSecondsOf(span: Span): number {
    return Math.round(Number(span.days) * DAY + Number(span.hours) * HOUR)
}

function filled(span: Span): boolean {
    return [span.days, span.hours].every(value => value !== "" && Number.isFinite(Number(value)) && Number(value) >= 0) &&
        Number.isInteger(Number(span.days))
}

export type Draft = {breakBefore: Span, length: Span}

export function draftOf(scheduled: ScheduledSeason | undefined): Draft {
    if (scheduled === undefined) return {breakBefore: {days: "0", hours: "0"}, length: {days: "28", hours: "0"}}
    return {breakBefore: spanDraftOf(scheduled.breakBefore), length: spanDraftOf(scheduled.length)}
}

export function complete(draft: Draft): boolean {
    return filled(draft.breakBefore) && filled(draft.length) && spanSecondsOf(draft.length) > 0
}

// The window the durations give: the break after the season before's end; season 0's counts from its origin.
export function windowOf(draft: Draft, row: Row | undefined, rows: Row[]): {startsAt: Date, endsAt: Date} | undefined {
    if (!complete(draft)) return undefined
    const number = row?.number ?? (rows[rows.length - 1]?.number ?? -1) + 1
    const before = rows.find(other => other.number === number - 1)
    const origin = before?.endsAt ?? (row?.startsAt && new Date(row.startsAt.getTime() - secondsOf(row.scheduled.breakBefore) * 1000))
    if (origin === undefined) return undefined
    const startsAt = new Date(origin.getTime() + spanSecondsOf(draft.breakBefore) * 1000)
    return {startsAt, endsAt: new Date(startsAt.getTime() + spanSecondsOf(draft.length) * 1000)}
}

// A duration the form left as it was is sent as kept, to the nanosecond.
function durationOf(span: Span, kept: Duration | undefined): Duration {
    if (kept !== undefined && span.days === spanDraftOf(kept).days && span.hours === spanDraftOf(kept).hours) return kept
    return new Duration({seconds: BigInt(spanSecondsOf(span))})
}

export function questionOf(draft: Draft, row: Row | undefined, rows: Row[]): string {
    const window = windowOf(draft, row, rows)
    const span = `a break of ${spanOf(durationOf(draft.breakBefore, undefined))}, then ${spanOf(durationOf(draft.length, undefined))}`
    const dates = window === undefined ? "" : `, from ${window.startsAt.toLocaleString()} to ${window.endsAt.toLocaleString()}`
    return row === undefined ? `Add a season: ${span}${dates}?` : `Run season ${row.number}: ${span}${dates}?`
}

export async function addSeason(gamemaster: GamemasterAdmin, draft: Draft): Promise<number> {
    const added = await gamemaster.addSeason({
        breakBefore: durationOf(draft.breakBefore, undefined), length: durationOf(draft.length, undefined),
    })
    return added.number
}

export async function editSeason(gamemaster: GamemasterAdmin, row: Row, draft: Draft): Promise<void> {
    await gamemaster.editSeason({
        number: row.number, breakBefore: durationOf(draft.breakBefore, row.scheduled.breakBefore),
        length: durationOf(draft.length, row.scheduled.length),
    })
}

export async function deleteSeason(gamemaster: GamemasterAdmin, number: number): Promise<void> {
    await gamemaster.deleteSeason({number})
}
