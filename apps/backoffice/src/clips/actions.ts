import {Timestamp} from "@bufbuild/protobuf"
import {LONG_CALL_MS} from "../api.ts"
import {ClipsAdmin} from "../clients.ts"
import {StatsSnapshot} from "../gen/grpc/clips/v1/admin_pb.ts"
import {csvOf, tableOf} from "./table.ts"

export type Window = {since: Date | undefined, until: Date | undefined}

export type Span = "last" | "week" | "month" | "all"

const DAY_MS = 24 * 60 * 60 * 1000

// The window of the snapshots asked: the last one alone is from its own time on.
export function windowOf(span: Span, latest: StatsSnapshot | undefined, now: Date): Window {
    switch (span) {
        case "last":
            return {since: latest?.takenAt?.toDate(), until: undefined}
        case "week":
            return {since: new Date(now.getTime() - 7 * DAY_MS), until: undefined}
        case "month":
            return {since: new Date(now.getTime() - 30 * DAY_MS), until: undefined}
        case "all":
            return {since: undefined, until: undefined}
    }
}

export type Table = {name: string, csv: string, rows: number}

// Every row of the snapshots taken in the window, page after page, as the backend keeps them.
export async function documentsOf(clips: ClipsAdmin, window: Window, signal?: AbortSignal): Promise<string[]> {
    const documents: string[] = []
    let page = ""
    do {
        const answer = await clips.listRows({
            since: window.since === undefined ? undefined : Timestamp.fromDate(window.since),
            until: window.until === undefined ? undefined : Timestamp.fromDate(window.until),
            limit: 2000,
            page,
        }, {signal, timeoutMs: LONG_CALL_MS})
        documents.push(...answer.rows.map(row => row.document))
        page = answer.nextPage
    } while (page !== "")
    return documents
}

export async function tableFileOf(clips: ClipsAdmin, window: Window, now: Date): Promise<Table> {
    const documents = await documentsOf(clips, window)
    return {
        name: `clip-stats-${stampOf(window.since)}-${stampOf(window.until ?? now)}.csv`,
        csv: csvOf(tableOf(documents)),
        rows: documents.length,
    }
}

// 2026-10-18T06:00:00.000Z is 20261018T0600Z: a file name every system keeps. No start is "start".
function stampOf(at: Date | undefined): string {
    return at === undefined ? "start" : at.toISOString().slice(0, 16).replace(/[-:]/g, "") + "Z"
}
