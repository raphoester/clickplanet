import {describe, expect, it} from "vitest"
import {paceOf, playedAt, PODIUM_ENDING_SECONDS, timelineFor, timelineOf} from "./pace.ts"

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

    it("is the same in the control arm of the length experiment as with no arm, whatever happens", () => {
        for (const screens of [0, 0.4, 1, 2.5, 5, 7.5, 100]) {
            for (const ending of [undefined, PODIUM_ENDING_SECONDS]) {
                expect(timelineOf(screens, undefined, ending, "as-chosen")).toEqual(timelineOf(screens, undefined, ending))
            }
        }
    })

    it("is the shortest a front plays in the short arm, however far its fighting moves", () => {
        expect(timelineOf(5, undefined, undefined, "short")).toEqual({seconds: 8.5, ending: 2.5})
        expect(timelineOf(100, undefined, undefined, "short")).toEqual(timelineOf(0))
        expect(timelineOf(5, undefined, PODIUM_ENDING_SECONDS, "short")).toEqual({seconds: 10, ending: PODIUM_ENDING_SECONDS})
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
