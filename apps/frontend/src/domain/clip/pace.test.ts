import {describe, expect, it} from "vitest"
import {blendedPaceOf, paceOf, playedAt, timelineFor, timelineOf} from "./pace.ts"

describe("the pace of a replay", () => {
    const quietThenBusy = [...Array.from({length: 90}, (_, i) => 900 + i), 10, 20, 30]

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
})

describe("the length of a clip", () => {
    it("is short when the action stays in one place, however long it went on", () => {
        expect(timelineOf(0).seconds).toBe(8.5)
    })

    it("grows with how far the action moves, up to what a feed holds a viewer for", () => {
        expect(timelineOf(5).seconds).toBe(14.5)
        expect(timelineOf(100).seconds).toBe(17.5)
    })

    it("is as long as it plays, and the call to act", () => {
        expect(timelineFor(16.1).seconds).toBe(18.5)
    })

    it("is what it is told, never past 22", () => {
        expect(timelineOf(0, 20).seconds).toBe(20)
        expect(timelineOf(0, 40).seconds).toBe(22)
    })

    it("plays the whole replay before its ending, then holds", () => {
        const timeline = {seconds: 20, ending: 2.5}

        expect(playedAt(timeline, 0)).toBe(0)
        expect(playedAt(timeline, 8.75)).toBeCloseTo(0.5)
        expect(playedAt(timeline, 19)).toBe(1)
    })
})

describe("a saga's pace", () => {
    const HOUR = 3_600_000
    const MINUTE = 60_000
    // A day with nothing until one busy hour at its end.
    const pace = blendedPaceOf([23 * HOUR, 23 * HOUR + 1, 23 * HOUR + 2], 0, 24 * HOUR, 0.4, 1)

    it("spends part of the play on the calendar, so a quiet day still goes by on the clock", () => {
        expect(pace.shareOf(0)).toBe(0)
        expect(pace.shareOf(12 * HOUR)).toBeCloseTo(0.2)
        expect(pace.shareOf(24 * HOUR)).toBe(1)
    })

    it("spends the rest on what happens", () => {
        expect(pace.shareOf(23 * HOUR + 2) - pace.shareOf(23 * HOUR - 1)).toBeGreaterThan(0.6)
    })

    it("finds the time a share of the play is at", () => {
        expect(pace.timeAt(0.2)).toBeCloseTo(12 * HOUR, -1)
        expect(pace.timeAt(pace.shareOf(23 * HOUR + 1))).toBe(23 * HOUR + 1)
    })

    it("runs into a thousand tiles enclosed at once, never standing still on them", () => {
        const enclosed = 12 * HOUR
        const run = blendedPaceOf(Array.from({length: 1000}, () => enclosed), 0, 24 * HOUR, 0.4, 5 * MINUTE)
        const before = run.shareOf(enclosed - 5 * MINUTE)

        expect(run.shareOf(enclosed) - before).toBeCloseTo(0.6)
        const times = [0.25, 0.5, 0.75].map((part) => run.timeAt(before + part * (run.shareOf(enclosed) - before)))
        expect(times[0]).toBeGreaterThan(enclosed - 5 * MINUTE)
        expect(times[1]).toBeGreaterThan(times[0])
        expect(times[2]).toBeGreaterThan(times[1])
        expect(times[2]).toBeLessThan(enclosed)
    })
})
