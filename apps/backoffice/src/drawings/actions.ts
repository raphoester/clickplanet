import {Timestamp} from "@bufbuild/protobuf"
import {PlanetAdmin, PlayerAdmin, SupportAdmin} from "../clients.ts"
import {Player} from "../gen/grpc/planet/v1/admin_pb.ts"
import {PlaceReport, ReportOutcome} from "../gen/grpc/support/v1/admin_pb.ts"
import {Named, namesOf} from "../players/actions.ts"
import {wordsOf} from "../refusal.ts"
import {Decided} from "../reports/actions.ts"

// A bomb's blast, and the most the server takes.
export const DEFAULT_RINGS = 8
export const MAX_RINGS = 50
// How long before its report a drawing is taken to have started.
export const BEFORE_REPORT_MS = 60 * 60 * 1000

export const MAP_URL = "https://clickplanet.lol/"

export type Restored = {
    tiles: number
    changed: number
    restored: number
    painters: Player[]
    paintersTotal: number
    keptSince: Date | undefined
}
export type Painters = Restored & Named
export type Cleared = {tiles: number, owned: number, cleared: number}

// The game, close up on the tile.
export function mapLinkOf(tileId: number): string {
    return `${MAP_URL}?tile=${tileId}`
}

export function beforeOf(report: PlaceReport): Date {
    const reported = report.createdAt?.toDate() ?? new Date()
    return new Date(reported.getTime() - BEFORE_REPORT_MS)
}

export async function listPlaceReports(support: SupportAdmin, openOnly: boolean, signal?: AbortSignal): Promise<PlaceReport[]> {
    const answer = await support.listPlaceReports({openOnly}, {signal})
    return answer.reports
}

export async function decidePlace(
    support: SupportAdmin, reportId: string, outcome: ReportOutcome, comment: string,
): Promise<Decided> {
    const said = comment.trim()
    const answer = await support.decidePlaceReport({reportId, outcome, comment: said})
    return {outcome, comment: said, decidedAt: dateOf(answer.decidedAt)}
}

export async function restoreArea(
    planet: PlanetAdmin, tileId: number, rings: number, before: Date, dryRun: boolean,
): Promise<Restored> {
    const answer = await planet.restoreArea({tileId, rings, before: Timestamp.fromDate(before), dryRun})
    return {
        tiles: answer.tiles,
        changed: answer.changed,
        restored: answer.restored,
        painters: answer.painters,
        paintersTotal: answer.paintersTotal,
        keptSince: answer.keptSince?.toDate(),
    }
}

// The dry run, with who each painter is: the names the players list shows.
export async function paintersOf(
    planet: PlanetAdmin, player: PlayerAdmin, tileId: number, rings: number, before: Date,
): Promise<Painters> {
    const counted = await restoreArea(planet, tileId, rings, before, true)
    return {...counted, ...await namesOf(player, counted.painters.map(painter => painter.accountId))}
}

export async function clearArea(planet: PlanetAdmin, tileId: number, rings: number, dryRun: boolean): Promise<Cleared> {
    const answer = await planet.clearArea({tileId, rings, dryRun})
    return {tiles: answer.tiles, owned: answer.owned, cleared: answer.cleared}
}

export type Action = "restore" | "clear" | "decide" | "list"

export const OLDER_THAN_THE_LEDGER = "The ledger does not reach back to then: clear the area instead."
export const PLACE_REPORT_GONE = "This report is no longer kept."

export function refusalOf(action: Action, err: unknown): string {
    switch (action) {
        case "restore":
            return wordsOf(err, {failedPrecondition: OLDER_THAN_THE_LEDGER})
        case "decide":
            return wordsOf(err, {notFound: PLACE_REPORT_GONE})
        case "clear":
        case "list":
            return wordsOf(err)
    }
}

// A date as a datetime-local input holds it, in the operator's time zone, to the minute.
export function localInputOf(date: Date): string {
    const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000)
    return local.toISOString().slice(0, 16)
}

// What a datetime-local input holds, as a date; undefined while it holds none.
export function dateOfInput(value: string): Date | undefined {
    if (value === "") return undefined
    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? undefined : date
}

function dateOf(at: {toDate(): Date} | undefined): Date {
    if (at === undefined) throw new Error("the server answered no time")
    return at.toDate()
}
