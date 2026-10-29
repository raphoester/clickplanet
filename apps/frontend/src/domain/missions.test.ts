import {describe, expect, it} from "vitest"
import {DayMissions, FlagMission, Mission} from "../backends/missions.ts"
import {awardOf, newlyDoneFlagMission, newlyPaid, orderOf, parseSeenMissions, reachedOf, seenNow, timeLeft, unseen} from "./missions.ts"

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
        expect(orderOf({kind: "takeFromIn", country: "dz", ground: "br", goal: 60, taken: 0})).toBe("Take 60 tiles from")
    })

    it("says what the other kinds ask, one or many", () => {
        expect(orderOf({kind: "catchBoxes", goal: 4, reached: 0})).toBe("Catch 4 boxes")
        expect(orderOf({kind: "answerQuizzes", goal: 1, reached: 0})).toBe("Answer a quiz right")
        expect(orderOf({kind: "bombIn", country: "br", goal: 1, reached: 0})).toBe("Drop a bomb on")
        expect(orderOf({kind: "bombIn", country: "br", goal: 2, reached: 0})).toBe("Drop 2 bombs on")
        expect(orderOf({kind: "shieldTiles", shields: 10, goal: 2, reached: 0})).toBe("Put 10 shields on 2 tiles")
        expect(orderOf({kind: "fortifyTiles", goal: 40, reached: 0})).toBe("Fortify 40 tiles")
    })
})

describe("reachedOf", () => {
    it("is how far an order went, never past its goal", () => {
        expect(reachedOf({kind: "takeFromIn", country: "dz", ground: "br", goal: 60, taken: 12})).toBe(12)
        expect(reachedOf({kind: "catchBoxes", goal: 4, reached: 5})).toBe(4)
    })
})

describe("unseen", () => {
    it("counts each mission drawn, done or paid since it was last seen", () => {
        const seen = seenNow([mission({id: "bank"}), mission({id: "rival", done: true}), mission({id: "ground", done: true})])

        expect(unseen([
            mission({id: "bank"}),
            mission({id: "rival", done: true, paid: true}),
            mission({id: "ground", done: true}),
            mission({id: "tomorrow"}),
        ], seen)).toBe(2)
    })

    it("counts a whole day not seen yet", () => {
        expect(unseen([mission({id: "bank"}), mission({id: "rival"}), mission({id: "ground"})], {})).toBe(3)
    })
})

describe("seenNow", () => {
    it("keeps only the missions shown, each at its stage", () => {
        expect(seenNow([mission({id: "bank", done: true, paid: true}), mission({id: "rival", done: true}), mission({id: "ground"})]))
            .toEqual({bank: "paid", rival: "done", ground: "new"})
    })
})

describe("parseSeenMissions", () => {
    it("reads what was stored, and nothing from what is not", () => {
        expect(parseSeenMissions(JSON.stringify({bank: "paid", rival: "lost", ground: 3}))).toEqual({bank: "paid"})
        for (const raw of [null, "", "{", "[]", "7"]) expect(parseSeenMissions(raw)).toEqual({})
    })
})

describe("newlyDoneFlagMission", () => {
    const flag = (overrides: Partial<FlagMission> = {}): FlagMission => ({
        id: "flag-22",
        order: {kind: "breakShields", country: "pw", goal: 450, reached: 449},
        points: 2,
        done: false,
        ...overrides,
    })
    const day = (flagMission?: FlagMission): DayMissions => ({endsAt: 0, missions: [], flagMission})

    it("is the flag's mission read not done and done now", () => {
        const done = flag({done: true})

        expect(newlyDoneFlagMission(day(flag()), day(done))).toBe(done)
    })

    it("is none when it was done already, is not done yet, or is another day's", () => {
        expect(newlyDoneFlagMission(day(flag({done: true})), day(flag({done: true})))).toBeUndefined()
        expect(newlyDoneFlagMission(day(flag()), day(flag()))).toBeUndefined()
        expect(newlyDoneFlagMission(day(flag()), day(flag({id: "flag-23", done: true})))).toBeUndefined()
        expect(newlyDoneFlagMission(day(), day(flag({done: true})))).toBeUndefined()
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
