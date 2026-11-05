import {describe, expect, it} from "vitest"
import {DayMissions, FlagMission, Mission, MissionOrder} from "../backends/missions.ts"
import {movedOf, nearestDone, stillOpen} from "./missionProgress.ts"

function mission(id: string, order: MissionOrder, done = false): Mission {
    return {id, order, reward: {kind: "bomb", amount: 1}, done, paid: done}
}

function dayOf(missions: Mission[], flagMission?: FlagMission): DayMissions {
    return {endsAt: 0, missions, flagMission}
}

const rival = (taken: number, done = false) => mission("rival", {kind: "takeFrom", country: "ie", goal: 80, taken}, done)
const ground = (taken: number) => mission("ground", {kind: "takeIn", country: "kz", goal: 300, taken})
const bank = (taken: number) => mission("bank", {kind: "takeFor", country: "fr", goal: 120, taken})
const boxes = (reached: number) => mission("boxes", {kind: "catchBoxes", goal: 4, reached})
const shields = (reached: number, done = false): FlagMission =>
    ({id: "flag", order: {kind: "breakShields", country: "ie", goal: 30, reached}, points: 5, done})

describe("movedOf", () => {
    it("names the missions a day moved, a box caught as a tile taken", () => {
        const moved = movedOf(dayOf([rival(3), ground(10), boxes(1)]), dayOf([rival(4), ground(10), boxes(2)]))

        expect(moved.map(({id}) => id)).toEqual(["rival", "boxes"])
    })

    it("leaves out the bank for the flag, which every take moves", () => {
        expect(movedOf(dayOf([bank(5)]), dayOf([bank(6)]))).toEqual([])
    })

    it("leaves out a mission done: its card says it", () => {
        expect(movedOf(dayOf([rival(79)]), dayOf([rival(80, true)]))).toEqual([])
    })

    it("names the flag's mission when it moved", () => {
        const moved = movedOf(dayOf([], shields(3)), dayOf([], shields(4)))

        expect(moved).toEqual([{id: "flag", order: shields(4).order, flag: true}])
    })

    it("names nothing on another day", () => {
        expect(movedOf(dayOf([rival(3)]), dayOf([{...rival(9), id: "tomorrow"}]))).toEqual([])
    })
})

describe("nearestDone", () => {
    it("takes the mission nearest its goal", () => {
        const moved = movedOf(dayOf([rival(3), ground(10)]), dayOf([rival(4), ground(290)]))

        expect(nearestDone(moved)?.id).toBe("ground")
    })

    it("has none when nothing moved", () => {
        expect(nearestDone([])).toBeUndefined()
    })
})

describe("stillOpen", () => {
    it("is false once the mission is done, or gone with its day", () => {
        expect(stillOpen("rival", dayOf([rival(4)]))).toBe(true)
        expect(stillOpen("rival", dayOf([rival(80, true)]))).toBe(false)
        expect(stillOpen("flag", dayOf([], shields(30, true)))).toBe(false)
        expect(stillOpen("rival", dayOf([]))).toBe(false)
    })
})
