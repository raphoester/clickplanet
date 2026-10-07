import {describe, expect, it} from "vitest"
import {bombShareOf, momentsOf, paceOf, playedAt, timelineOf} from "./pace.ts"

describe("the pace of a replay", () => {
    const quietThenBusy = momentsOf([...Array.from({length: 90}, (_, i) => 900 + i), 10, 20, 30], [], 0)

    it("spends the clip on what happens and none of it on the quiet between", () => {
        const pace = paceOf(quietThenBusy, 0, 1000)

        expect(pace.timeAt(0)).toBe(10)
        expect(pace.timeAt(0.03)).toBe(30)
        expect(pace.timeAt(0.04)).toBe(900)
        expect(pace.timeAt(1)).toBe(989)
        expect(pace.shareOf(800)).toBeCloseTo(3 / 93)
        expect(pace.shareOf(1000)).toBe(1)
    })

    it("plays evenly when nothing happened", () => {
        expect(paceOf([], 0, 1000).timeAt(0.5)).toBeCloseTo(500)
    })

    it("comes back to about the share it was asked for, the share moving in steps", () => {
        const pace = paceOf(quietThenBusy, 0, 1000)

        expect(pace.shareOf(pace.timeAt(0.42))).toBeGreaterThanOrEqual(0.42)
        expect(pace.shareOf(pace.timeAt(0.42))).toBeLessThanOrEqual(0.42 + 1 / 93)
    })

    it("holds still on a bomb for the share of the clip it is given", () => {
        const changes = Array.from({length: 100}, (_, i) => i * 10)
        const pace = paceOf(momentsOf(changes, [505], 0.1), 0, 1000)
        const held = pace.heldOn(505)

        expect(held.to - held.from).toBeCloseTo(0.1)
        expect(pace.timeAt((held.from + held.to) / 2)).toBe(505)
    })
})

describe("the length of a clip", () => {
    it("is short when the action stays in one place, however long it went on", () => {
        expect(timelineOf(0, 0).seconds).toBe(7.5)
    })

    it("grows with how far the action moves, up to what a feed holds a viewer for", () => {
        expect(timelineOf(5, 0).seconds).toBe(13.5)
        expect(timelineOf(100, 0).seconds).toBe(17.5)
    })

    it("gives every bomb the seconds it takes to go off, and never runs past 22", () => {
        expect(timelineOf(0, 2).seconds).toBe(11.5)
        expect(bombShareOf(timelineOf(0, 2))).toBeCloseTo(2 / 9)
        expect(timelineOf(100, 12).seconds).toBe(22)
    })

    it("is what it is told", () => {
        expect(timelineOf(0, 0, 20).seconds).toBe(20)
    })

    it("plays the whole replay before its ending, then holds", () => {
        const timeline = {seconds: 20, ending: 2.5}

        expect(playedAt(timeline, 0)).toBe(0)
        expect(playedAt(timeline, 8.75)).toBeCloseTo(0.5)
        expect(playedAt(timeline, 19)).toBe(1)
    })
})
