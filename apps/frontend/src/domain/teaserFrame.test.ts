import {describe, expect, it} from "vitest"
import {teaserFrameOf} from "./teaserFrame.ts"

describe("teaserFrameOf", () => {
    it("writes the day out and the time in UTC, whatever the machine's time zone", () => {
        expect(teaserFrameOf(new Date("2026-11-07T20:00:00Z"), false))
            .toEqual({name: "Battle", day: "Saturday 7 November", time: "20:00 UTC"})
    })

    it("names a season's finale as the game does", () => {
        expect(teaserFrameOf(new Date("2026-11-28T17:05:00Z"), true))
            .toEqual({name: "Final Battle", day: "Saturday 28 November", time: "17:05 UTC"})
    })

    it("keeps a day that has just begun in UTC", () => {
        expect(teaserFrameOf(new Date("2026-11-09T00:30:00Z"), false))
            .toEqual({name: "Battle", day: "Monday 9 November", time: "00:30 UTC"})
    })
})
