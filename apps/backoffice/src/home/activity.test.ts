import {Duration, Timestamp} from "@bufbuild/protobuf"
import {describe, expect, it} from "vitest"
import {ActivitySlot, CountryActivity, GetActivityResponse} from "../gen/grpc/planet/v1/admin_pb.ts"
import {countriesBy, labelled, labelEveryOf, Slot, slotsOf, ticksOf, totalsOf} from "./activity.ts"

const at = new Date("2026-10-23T12:00:00Z")

function slotAt(start: Date): Slot {
    return {start, end: start, clicks: 0, tiles: 0, players: 0, filling: false}
}

describe("ticksOf", () => {
    it("steps by 1, 2 or 5 a power of ten, up to the highest value or past it", () => {
        expect(ticksOf(1234)).toEqual([0, 500, 1000, 1500])
        expect(ticksOf(7)).toEqual([0, 5, 10])
        expect(ticksOf(3)).toEqual([0, 1, 2, 3])
    })

    it("never steps under one, since every chart counts", () => {
        expect(ticksOf(1)).toEqual([0, 1])
        expect(ticksOf(0)).toEqual([0, 1])
    })
})

describe("slotsOf", () => {
    it("times each slot from the start of the first, and marks the one still filling", () => {
        const slots = slotsOf(new GetActivityResponse({
            since: Timestamp.fromDate(at),
            until: Timestamp.fromDate(new Date(at.getTime() + 20 * 60_000)),
            step: new Duration({seconds: 900n}),
            slots: [new ActivitySlot({clicks: 4, tiles: 5, players: 2}), new ActivitySlot({clicks: 1})],
        }))

        expect(slots).toEqual([
            {start: at, end: new Date(at.getTime() + 900_000), clicks: 4, tiles: 5, players: 2, filling: false},
            {
                start: new Date(at.getTime() + 900_000), end: new Date(at.getTime() + 1_800_000),
                clicks: 1, tiles: 0, players: 0, filling: true,
            },
        ])
        expect(totalsOf(slots)).toEqual({clicks: 5, tiles: 5})
    })
})

describe("the time labels", () => {
    it("fall every hour over 6 hours, every 4 over a day and every 12 over 3 days", () => {
        expect(labelEveryOf(5 * 60_000)).toBe(1)
        expect(labelEveryOf(15 * 60_000)).toBe(4)
        expect(labelEveryOf(3600_000)).toBe(12)
    })

    it("mark the slots that start on such an hour", () => {
        expect(labelled(slotAt(new Date(2026, 9, 23, 8, 0)), 4)).toBe(true)
        expect(labelled(slotAt(new Date(2026, 9, 23, 9, 0)), 4)).toBe(false)
        expect(labelled(slotAt(new Date(2026, 9, 23, 8, 15)), 4)).toBe(false)
    })
})

describe("countriesBy", () => {
    const countries = [
        new CountryActivity({countryId: "fr", held: 100, won: 5, lost: 9}),
        new CountryActivity({countryId: "dz", held: 50, won: 30, lost: 1}),
        new CountryActivity({countryId: "pl", held: 10, won: 30}),
    ]

    it("keeps the server's order, most held first, and orders by won or lost on demand", () => {
        expect(countriesBy(countries, "held").map(country => country.countryId)).toEqual(["fr", "dz", "pl"])
        expect(countriesBy(countries, "won").map(country => country.countryId)).toEqual(["dz", "pl", "fr"])
        expect(countriesBy(countries, "lost").map(country => country.countryId)).toEqual(["fr", "dz", "pl"])
    })
})
