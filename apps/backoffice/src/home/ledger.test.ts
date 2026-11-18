import {Timestamp} from "@bufbuild/protobuf"
import {describe, expect, it} from "vitest"
import {AT, FakeAdmin} from "../fakeAdmin.ts"
import {LedgerChange} from "../gen/grpc/planet/v1/ledger_pb.ts"
import {addsTo, boardOfPages, caughtUp, changesOf, countriesBy, countriesOf, playersOf, sinceOf, slotsOf, windowOf} from "./ledger.ts"

const MINUTE = 60_000

describe("windowOf", () => {
    it("ends on the slot still filling, and starts as many slots back as the span holds", () => {
        const until = new Date("2026-10-23T12:07:00Z")

        expect(windowOf(until, 1, 5 * MINUTE)).toEqual({since: new Date("2026-10-23T11:10:00Z"), until, count: 12, step: 5 * MINUTE})
        expect(windowOf(until, 24, 15 * MINUTE).since).toEqual(new Date("2026-10-22T12:15:00Z"))
        expect(windowOf(until, 24, 15 * MINUTE).count).toBe(96)
    })
})

describe("reading the ledger in parts", () => {
    const first = windowOf(new Date("2026-10-23T12:07:00Z"), 1, 5 * MINUTE)
    const next = windowOf(new Date("2026-10-23T12:12:30Z"), 1, 5 * MINUTE)
    const at = (iso: string) => new LedgerChange({at: Timestamp.fromDate(new Date(iso)), by: iso})

    it("reads the whole window first, then from a minute before the end of what it kept", () => {
        expect(sinceOf(undefined, first)).toEqual(first.since)
        expect(sinceOf({window: first, changes: [], board: "earth-1"}, next)).toEqual(new Date("2026-10-23T12:06:00Z"))
    })

    it("reads the whole window again for other slots, or after a pause longer than the window", () => {
        const day = windowOf(new Date("2026-10-23T12:07:00Z"), 24, 15 * MINUTE)
        const later = windowOf(new Date("2026-10-23T15:00:00Z"), 1, 5 * MINUTE)

        expect(sinceOf({window: day, changes: [], board: "earth-1"}, next)).toEqual(next.since)
        expect(sinceOf({window: first, changes: [], board: "earth-1"}, later)).toEqual(later.since)
    })

    it("keeps what it read before the read's start and inside the window, then adds what it read", () => {
        const kept = {
            window: first, board: "earth-1",
            changes: [at("2026-10-23T11:12:00Z"), at("2026-10-23T12:05:59Z"), at("2026-10-23T12:06:30Z")],
        }
        const since = sinceOf(kept, next)

        const ledger = caughtUp(kept, next, since, [at("2026-10-23T12:06:30Z"), at("2026-10-23T12:12:00Z")], "earth-1")

        expect(ledger.window).toBe(next)
        expect(ledger.board).toBe("earth-1")
        expect(ledger.changes.map(change => change.by)).toEqual([
            "2026-10-23T12:05:59Z", "2026-10-23T12:06:30Z", "2026-10-23T12:12:00Z",
        ])
    })

    it("adds a read to what it kept only on the same board: after a switch it reads the window whole", () => {
        const kept = {window: first, changes: [at("2026-10-23T12:05:59Z")], board: "earth-1"}
        const since = sinceOf(kept, next)

        expect(addsTo(kept, next, since, "earth-1")).toBe(true)
        expect(addsTo(kept, next, since, "fixture-1")).toBe(false)
        expect(addsTo(kept, next, next.since, "fixture-1")).toBe(true)
        expect(addsTo(undefined, next, next.since, "fixture-1")).toBe(true)
        // Nothing of the old board is kept, whatever the read's start.
        const whole = caughtUp(kept, next, next.since, [at("2026-10-23T12:12:00Z")], "fixture-1")
        expect(whole.board).toBe("fixture-1")
        expect(whole.changes.map(change => change.by)).toEqual(["2026-10-23T12:12:00Z"])
        expect(caughtUp(kept, next, since, [], "fixture-1").changes).toEqual([])
    })

    it("names the board of a read whose pages all name one, and none when the world switched between two", () => {
        expect(boardOfPages(["earth-1"])).toBe("earth-1")
        expect(boardOfPages(["earth-1", "earth-1", "earth-1"])).toBe("earth-1")
        expect(boardOfPages(["earth-1", "fixture-1"])).toBeUndefined()
        expect(boardOfPages([])).toBeUndefined()
    })
})

describe("the ledger's counts", () => {
    // The fake's hour: BraveFox42 takes three tiles and strikes a shield, and a guest's spread takes one back.
    const ledger = new FakeAdmin().ledger

    it("puts each change in its slot: a click is counted, and a tile that changed hands counts its taker", () => {
        const slots = slotsOf(ledger, {since: new Date(AT.getTime() - 60 * MINUTE), until: AT, count: 3, step: 2 * MINUTE})

        expect(slots.map(slot => [slot.clicks, slot.tiles, slot.players])).toEqual([
            [1, 1, 1],
            [1, 2, 2],
            [2, 1, 1],
        ])
        expect(slots.map(slot => slot.filling)).toEqual([false, false, false])
    })

    it("counts a player once on every address, and an address with no account as one", () => {
        expect(playersOf(ledger)).toBe(2)
    })

    it("counts how often each tile changed hands, a strike left out", () => {
        expect([...changesOf(ledger, 4)]).toEqual([0, 1, 2, 0, 1])
    })

    it("joins what each country holds now with what it won and lost", () => {
        const countries = countriesOf(ledger, new FakeAdmin().territories.territories)

        expect(countries).toEqual([
            {countryId: "fr", held: 1200, won: 3, lost: 1},
            {countryId: "dz", held: 800, won: 1, lost: 2},
        ])
        expect(countriesBy(countries, "lost").map(country => country.countryId)).toEqual(["dz", "fr"])
        expect(countriesBy(countries, "won").map(country => country.countryId)).toEqual(["fr", "dz"])
    })
})
