import {describe, expect, it} from "vitest"
import {Mission} from "../backends/missions.ts"
import {awardOf, collectable, done, orderOf, timeLeft} from "./missions.ts"

const mission = (overrides: Partial<Mission> = {}): Mission => ({
    slot: 1,
    kind: "takeFor",
    target: "fr",
    goal: 60,
    progress: 0,
    collected: false,
    reward: {kind: "shields", amount: 10},
    ...overrides,
})

describe("orderOf", () => {
    it("says how many tiles to take and how they relate to the country named after it", () => {
        expect(orderOf(mission())).toBe("Take 60 tiles for")
        expect(orderOf(mission({kind: "takeFrom", goal: 40}))).toBe("Take 40 tiles from")
        expect(orderOf(mission({kind: "takeIn", goal: 200}))).toBe("Take 200 tiles in")
    })
})

describe("collectable", () => {
    it("counts the missions done and not collected", () => {
        expect(collectable([
            mission({slot: 1, progress: 60}),
            mission({slot: 2, progress: 60, collected: true}),
            mission({slot: 3, progress: 59}),
        ])).toBe(1)
        expect(done(mission({progress: 59}))).toBe(false)
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
