import {describe, expect, it} from "vitest"
import {Mission} from "../backends/missions.ts"
import {awardOf, newlyPaid, orderOf, timeLeft, waiting} from "./missions.ts"

const mission = (overrides: Partial<Mission> = {}): Mission => ({
    id: "bank",
    order: {kind: "takeFor", country: "fr", goal: 60, taken: 0},
    reward: {kind: "shields", amount: 10},
    done: false,
    paid: false,
    ...overrides,
})

describe("orderOf", () => {
    it("says how many tiles to take and how they relate to the country named after it", () => {
        expect(orderOf({kind: "takeFor", country: "fr", goal: 60, taken: 0})).toBe("Take 60 tiles for")
        expect(orderOf({kind: "takeFrom", country: "dz", goal: 40, taken: 0})).toBe("Take 40 tiles from")
        expect(orderOf({kind: "takeIn", country: "br", goal: 200, taken: 0})).toBe("Take 200 tiles in")
    })
})

describe("waiting", () => {
    it("counts the missions done and not paid yet", () => {
        expect(waiting([
            mission({id: "bank", done: true}),
            mission({id: "rival", done: true, paid: true}),
            mission({id: "ground"}),
        ])).toBe(1)
    })
})

describe("newlyPaid", () => {
    it("is what was read unpaid and is paid now", () => {
        const before = [mission({id: "bank", done: true}), mission({id: "rival", paid: true})]

        expect(newlyPaid(before, [
            mission({id: "bank", done: true, paid: true}),
            mission({id: "rival", paid: true}),
            mission({id: "ground", paid: true}),
        ]).map((paid) => paid.id)).toEqual(["bank"])
    })
})

describe("awardOf", () => {
    const rules = {blastRadius: 0.05, enclosureMaxTiles: 25}

    it("is the box a caught reward would show", () => {
        expect(awardOf({kind: "shields", amount: 7}, rules)).toEqual({kind: "shields", shields: 7})
        expect(awardOf({kind: "spreadClicks", amount: 4}, rules)).toEqual({kind: "spreadClicks", clicks: 4})
        expect(awardOf({kind: "bomb", amount: 1}, rules)).toEqual({kind: "bomb", radius: 0.05})
    })
})

describe("timeLeft", () => {
    const now = Date.UTC(2026, 9, 26, 12, 0)

    it("reads hours and minutes, then minutes alone", () => {
        expect(timeLeft(Date.UTC(2026, 9, 26, 21, 0), now)).toBe("9h 00m")
        expect(timeLeft(Date.UTC(2026, 9, 26, 12, 5, 30), now)).toBe("6m")
        expect(timeLeft(Date.UTC(2026, 9, 26, 13, 59, 30), now)).toBe("2h 00m")
        expect(timeLeft(now - 1, now)).toBe("0m")
    })
})
