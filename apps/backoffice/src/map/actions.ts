import {Timestamp} from "@bufbuild/protobuf"
import {LONG_CALL_MS} from "../api.ts"
import {PlanetAdmin} from "../clients.ts"
import {GetReplayResponse} from "../gen/grpc/planet/v1/admin_pb.ts"

export type Reassigned = {fromBefore: number, toBefore: number, moved: number, fromAfter: number, toAfter: number}

// Every tile `from` holds goes to `to`, paced on the server. A dry run counts both sides and moves nothing.
export async function reassign(planet: PlanetAdmin, from: string, to: string, dryRun: boolean): Promise<Reassigned> {
    const answer = await planet.reassignCountry(
        {fromCountryId: from, toCountryId: to, dryRun},
        {timeoutMs: LONG_CALL_MS},
    )
    return {
        fromBefore: answer.fromBefore,
        toBefore: answer.toBefore,
        moved: answer.moved,
        fromAfter: answer.fromAfter,
        toAfter: answer.toAfter,
    }
}

export type Painting = {flag: string, area: string, count: number, proximity: number}
export type Painted = {eligible: number, picked: number, painted: number, outsideArea: number}

// `count` tiles get `flag`, starting on `area`'s ground (anywhere with no area). A dry run draws and paints nothing.
export async function paint(planet: PlanetAdmin, painting: Painting, dryRun: boolean): Promise<Painted> {
    const answer = await planet.paintRandomTiles({
        flagCountryId: painting.flag,
        areaCountryId: painting.area,
        count: painting.count,
        proximity: painting.proximity,
        dryRun,
    }, {timeoutMs: LONG_CALL_MS})
    return {eligible: answer.eligible, picked: answer.picked, painted: answer.painted, outsideArea: answer.outsideArea}
}

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
