import {describe, expect, it} from "vitest"
import {labelled, labelEveryOf, Slot, ticksOf} from "./activity.ts"

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
