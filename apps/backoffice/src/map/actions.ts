import {Timestamp} from "@bufbuild/protobuf"
import {LONG_CALL_MS} from "../api.ts"
import {PlanetAdmin} from "../clients.ts"
import {timeOf} from "../components/time.ts"
import {GetReplayResponse} from "../gen/grpc/planet/v1/admin_pb.ts"
import {countOf} from "../support/labels.ts"

export type Window = {since: Date, until: Date | undefined}

// The ledger keeps 72 hours in memory: a replay reaches no further back.
export const RETENTION_HOURS = 72

export async function replay(planet: PlanetAdmin, window: Window, signal?: AbortSignal): Promise<GetReplayResponse> {
    return planet.getReplay({
        since: Timestamp.fromDate(window.since),
        until: window.until === undefined ? undefined : Timestamp.fromDate(window.until),
    }, {signal, timeoutMs: LONG_CALL_MS})
}

// The replay as `clips fetch` saves it, which the frontend's `npm run clip` reads: the whole answer, its eras with it,
// so each event is played on the board of the era in force at its time.
export function replayFileOf(answer: GetReplayResponse): {name: string, json: string} {
    return {name: `replay-${stampOf(answer.since)}-${stampOf(answer.until)}.json`, json: answer.toJsonString()}
}

// 2026-10-18T06:00:00.000Z is 20261018T0600Z: a file name every system keeps.
function stampOf(at: Timestamp | undefined): string {
    return (at?.toDate() ?? new Date(0)).toISOString().slice(0, 16).replace(/[-:]/g, "") + "Z"
}

// A stretch of the replay on one board: the era in force at its start, then each that started in it.
export type Era = {board: string, startedAt: Date | undefined, events: number}

// Each era the replay covers, oldest first, with the events played on its board: an event's tile ids are the board of
// the era in force at its time.
export function erasOf(answer: GetReplayResponse): Era[] {
    const eras: Era[] = answer.eras.map(era => ({board: era.board, startedAt: era.startedAt?.toDate(), events: 0}))
    for (const {at} of answer.events) {
        const time = at?.toDate().getTime() ?? 0
        let era = eras.length - 1
        while (era > 0 && (eras[era].startedAt?.getTime() ?? 0) > time) era--
        if (era >= 0) eras[era].events++
    }
    return eras
}

// The boards a replay is on, in words: "on earth-1", or each era with its start and its events, the cut between them.
export function boardsOf(eras: Era[]): string {
    if (eras.length === 0) return ""
    if (eras.length === 1) return `on ${eras[0].board}`
    return eras.map((era, i) => {
        const from = i === 0 || era.startedAt === undefined ? "" : ` from ${timeOf(era.startedAt)}`
        return `${i === 0 ? "on" : "then on"} ${era.board}${from} (${countOf(era.events, "event")})`
    }).join(", ")
}
