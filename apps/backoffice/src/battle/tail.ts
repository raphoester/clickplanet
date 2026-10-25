import {Timestamp} from "@bufbuild/protobuf"
import {LONG_CALL_MS} from "../api.ts"
import {PlanetAdmin} from "../clients.ts"
import {GetLedgerResponse, LedgerChange} from "../gen/grpc/planet/v1/admin_pb.ts"
import {Window} from "../home/ledger.ts"

// The ledger lags its last second in memory: a window that ends a little before now misses nothing.
export const LAG_MS = 5_000

export type Tail = {window: Window, ledger: GetLedgerResponse}

// Where the next read starts: where the last one stopped, unless the window now reaches back past it.
export function fromOf(last: Tail | undefined, window: Window): Date {
    if (last === undefined || last.window.since > window.since || last.window.until > window.until) return window.since
    return last.window.until
}

// The last tail and what was read from `from`, the changes before the window dropped.
export function joined(last: Tail | undefined, window: Window, from: Date, read: LedgerChange[]): Tail {
    const kept = last === undefined || from.getTime() === window.since.getTime() ? [] : last.ledger.changes
    const since = window.since.getTime()
    const first = kept.findIndex(change => millisOf(change.at) >= since)
    const changes = first < 0 ? read : [...kept.slice(first), ...read]
    return {window, ledger: new GetLedgerResponse({changes})}
}

export async function tailOf(planet: PlanetAdmin, last: Tail | undefined, window: Window, signal: AbortSignal): Promise<Tail> {
    const from = fromOf(last, window)
    if (from >= window.until) return joined(last, window, from, [])

    const answer = await planet.getLedger(
        {since: Timestamp.fromDate(from), until: Timestamp.fromDate(window.until)},
        {signal, timeoutMs: LONG_CALL_MS},
    )
    return joined(last, window, from, answer.changes)
}

function millisOf(at: Timestamp | undefined): number {
    return at === undefined ? 0 : Number(at.seconds) * 1000 + at.nanos / 1e6
}
