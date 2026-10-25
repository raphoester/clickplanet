import {Timestamp} from "@bufbuild/protobuf"
import {describe, expect, it} from "vitest"
import {AT, FakeAdmin} from "../fakeAdmin.ts"
import {LedgerChange} from "../gen/grpc/planet/v1/admin_pb.ts"
import {TileUpdate} from "../gen/grpc/planet/v1/planet_pb.ts"
import {windowOf} from "../home/ledger.ts"
import {MINUTE} from "./counters.ts"
import {fromOf, joined, tailOf} from "./tail.ts"

function changeAt(minutesBefore: number, tileId: number): LedgerChange {
    return new LedgerChange({
        at: Timestamp.fromDate(new Date(AT.getTime() - minutesBefore * MINUTE)),
        update: new TileUpdate({tileId, countryId: "fr", clicked: true}), by: "scope:203.0.113.7",
    })
}

const hour = windowOf(AT, 1, MINUTE)
const later = windowOf(new Date(AT.getTime() + 2 * MINUTE), 1, MINUTE)

// A request as the page sent it: its bounds are Timestamps.
type Sent = {since: {toDate(): Date}, until: {toDate(): Date}}

describe("the tail of the ledger", () => {
    it("reads a first window whole, then on from where the last read stopped", () => {
        const first = joined(undefined, hour, hour.since, [changeAt(30, 1)])

        expect(fromOf(undefined, hour)).toEqual(hour.since)
        expect(fromOf(first, later)).toEqual(AT)
        expect(fromOf(first, windowOf(AT, 2, MINUTE)), "a wider span reaches back past the last read").toEqual(windowOf(AT, 2, MINUTE).since)
    })

    it("drops the changes the window left behind and keeps the new ones after the old", () => {
        const first = joined(undefined, hour, hour.since, [changeAt(59, 1), changeAt(30, 2)])

        const next = joined(first, later, AT, [changeAt(-1, 3)])

        expect(next.ledger.changes.map(change => change.update?.tileId)).toEqual([2, 3])
        expect(next.window).toBe(later)
    })

    it("asks the ledger for the new minutes alone", async () => {
        const admin = new FakeAdmin()
        const first = await tailOf(admin, undefined, hour, new AbortController().signal)
        admin.ledger.changes = []

        await tailOf(admin, first, later, new AbortController().signal)

        const asked = admin.askedFor("getLedger") as Sent[]
        expect(asked.map(request => [request.since.toDate(), request.until.toDate()])).toEqual([
            [hour.since, hour.until],
            [hour.until, later.until],
        ])
    })

    it("asks nothing when no minute is new", async () => {
        const admin = new FakeAdmin()
        const first = await tailOf(admin, undefined, hour, new AbortController().signal)

        const same = await tailOf(admin, first, hour, new AbortController().signal)

        expect(admin.askedFor("getLedger")).toHaveLength(1)
        expect(same.ledger.changes).toHaveLength(first.ledger.changes.length)
    })
})
