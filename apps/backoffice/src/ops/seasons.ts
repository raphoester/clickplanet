import {Timestamp} from "@bufbuild/protobuf"
import {GamemasterAdmin} from "../clients.ts"
import {GetSeasonsResponse} from "../gen/grpc/gamemaster/v1/admin_pb.ts"
import {Season} from "../gen/grpc/gamemaster/v1/gamemaster_pb.ts"
import {localOf} from "./schedule.ts"

// A line of the seasons: a season and its times.
export type Row = {season: Season, startsAt?: Date, endsAt?: Date}

// The seasons by number.
export function rowsOf(seasons: GetSeasonsResponse): Row[] {
    return seasons.seasons
        .map(season => ({season, startsAt: season.startsAt?.toDate(), endsAt: season.endsAt?.toDate()}))
        .sort((a, b) => a.season.number - b.season.number)
}

export type Phase = "done" | "running" | "to come"

export function phaseOf(row: Row, now: Date): Phase {
    if (row.endsAt !== undefined && row.endsAt <= now) return "done"
    if (row.startsAt !== undefined && row.startsAt <= now) return "running"
    return "to come"
}

export function started(row: Row, now: Date): boolean {
    return phaseOf(row, now) !== "to come"
}

// Only the last season, and only before it starts.
export function deletable(row: Row, rows: Row[], now: Date): boolean {
    return row === rows[rows.length - 1] && !started(row, now)
}

// A season as the form edits it: each time a datetime-local, in the operator's own time.
export type Draft = {startsAt: string, endsAt: string}

// A new season starts where the last one ends.
export function draftOf(season: Season | undefined, rows: Row[]): Draft {
    if (season === undefined) return {startsAt: localOf(rows[rows.length - 1]?.endsAt), endsAt: ""}
    return {startsAt: localOf(season.startsAt?.toDate()), endsAt: localOf(season.endsAt?.toDate())}
}

export function complete(draft: Draft): boolean {
    return draft.startsAt !== "" && draft.endsAt !== ""
}

// A time the form left as it was is sent as it was kept, to the second: a started season's start must not move.
function timeOf(local: string, kept: Timestamp | undefined): Timestamp {
    if (kept !== undefined && local === localOf(kept.toDate())) return kept
    return Timestamp.fromDate(new Date(local))
}

export function questionOf(draft: Draft, season: Season | undefined): string {
    const window = `from ${new Date(draft.startsAt).toLocaleString()} to ${new Date(draft.endsAt).toLocaleString()}`
    return season === undefined ? `Add a season ${window}?` : `Run season ${season.number} ${window}?`
}

export async function addSeason(gamemaster: GamemasterAdmin, draft: Draft): Promise<number> {
    const added = await gamemaster.addSeason({
        startsAt: Timestamp.fromDate(new Date(draft.startsAt)), endsAt: Timestamp.fromDate(new Date(draft.endsAt)),
    })
    return added.number
}

export async function editSeason(gamemaster: GamemasterAdmin, season: Season, draft: Draft): Promise<void> {
    await gamemaster.editSeason({
        number: season.number, startsAt: timeOf(draft.startsAt, season.startsAt), endsAt: timeOf(draft.endsAt, season.endsAt),
    })
}

export async function deleteSeason(gamemaster: GamemasterAdmin, number: number): Promise<void> {
    await gamemaster.deleteSeason({number})
}
