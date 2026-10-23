import {describe, expect, it} from "vitest"
import {AT, FakeAdmin} from "../fakeAdmin.ts"
import {changesOf, countriesBy, countriesOf, playersOf, slotsOf, windowOf} from "./ledger.ts"

const MINUTE = 60_000

describe("windowOf", () => {
    it("ends on the slot still filling, and starts as many slots back as the span holds", () => {
        const until = new Date("2026-10-23T12:07:00Z")

        expect(windowOf(until, 1, 5 * MINUTE)).toEqual({since: new Date("2026-10-23T11:10:00Z"), until, count: 12, step: 5 * MINUTE})
        expect(windowOf(until, 24, 15 * MINUTE).since).toEqual(new Date("2026-10-22T12:15:00Z"))
        expect(windowOf(until, 24, 15 * MINUTE).count).toBe(96)
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
