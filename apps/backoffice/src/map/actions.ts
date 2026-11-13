import {Timestamp} from "@bufbuild/protobuf"
import {LONG_CALL_MS} from "../api.ts"
import {PlanetAdmin} from "../clients.ts"
import {GetReplayResponse} from "../gen/grpc/planet/v1/admin_pb.ts"

export type Window = {since: Date, until: Date | undefined}

// The ledger keeps 72 hours in memory: a replay reaches no further back.
export const RETENTION_HOURS = 72

export async function replay(planet: PlanetAdmin, window: Window, signal?: AbortSignal): Promise<GetReplayResponse> {
    return planet.getReplay({
        since: Timestamp.fromDate(window.since),
        until: window.until === undefined ? undefined : Timestamp.fromDate(window.until),
    }, {signal, timeoutMs: LONG_CALL_MS})
}

// The replay as `clips fetch` saves it, which the frontend's `npm run clip` reads.
export function replayFileOf(answer: GetReplayResponse): {name: string, json: string} {
    return {name: `replay-${stampOf(answer.since)}-${stampOf(answer.until)}.json`, json: answer.toJsonString()}
}

// 2026-10-18T06:00:00.000Z is 20261018T0600Z: a file name every system keeps.
function stampOf(at: Timestamp | undefined): string {
    return (at?.toDate() ?? new Date(0)).toISOString().slice(0, 16).replace(/[-:]/g, "") + "Z"
}
