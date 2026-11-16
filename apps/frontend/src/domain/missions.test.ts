import {describe, expect, it} from "vitest"
import {DayMissions, FlagMission, Mission} from "../backends/missions.ts"
import {newlyDoneFlagMission, newlyPaid, parseSeenMissions, reachedOf, seenNow, timeLeft, unseen, xpOf} from "./missions.ts"

const mission = (overrides: Partial<Mission> = {}): Mission => ({
    id: "bank",
    order: {kind: "takeFor", country: "fr", goal: 60, taken: 0},
    done: false,
    paid: false,
    ...overrides,
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

describe("xpOf", () => {
    const rates = {takeFrom: {xp: 3, per: 2}, catchBoxes: {xp: 50, per: 1}}

    it("is the goal times the kind's rate, floored as the server floors it", () => {
        expect(xpOf({kind: "takeFrom", country: "dz", goal: 80, taken: 0}, rates)).toBe(120)
        expect(xpOf({kind: "takeFrom", country: "dz", goal: 85, taken: 0}, rates)).toBe(127)
        expect(xpOf({kind: "catchBoxes", goal: 4, reached: 1}, rates)).toBe(200)
    })

    it("is unknown for a kind with no rate", () => {
        expect(xpOf({kind: "bombIn", country: "br", goal: 2, reached: 0}, rates)).toBeUndefined()
    })
})

describe("timeLeft", () => {
    const now = Date.UTC(2026, 9, 26, 12, 0)

    it("is the time left in the day, none once it ended", () => {
        expect(timeLeft(Date.UTC(2026, 9, 26, 21, 0), now)).toEqual({days: 0, hours: 9, minutes: 0, seconds: 0})
        expect(timeLeft(Date.UTC(2026, 9, 26, 12, 5, 30), now)).toEqual({days: 0, hours: 0, minutes: 5, seconds: 30})
        expect(timeLeft(now - 1, now)).toEqual({days: 0, hours: 0, minutes: 0, seconds: 0})
    })
})
