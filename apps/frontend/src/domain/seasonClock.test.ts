import {describe, expect, it} from "vitest"
import {countdownsOf, endClock, endsLine} from "./seasonClock.ts"

const SECOND = 1000
const MINUTE = 60 * SECOND
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR

const endsAt = Date.UTC(2026, 9, 31, 23)
const season = {number: 0, lastDayEndsAt: endsAt - 2 * HOUR, endsAt}
const lastDayEndsAt = season.lastDayEndsAt

const today = (now: number) => countdownsOf(season, now)[0]

describe("countdownsOf", () => {
    it("counts down to today's end and to the season's", () => {
        expect(countdownsOf(season, lastDayEndsAt - 3 * DAY - (13 * HOUR + 5 * MINUTE + 20 * SECOND))).toEqual([
            {label: "Today ends in", tag: "Today", left: "13h 05m 20s"},
            {label: "Season ends in", tag: "Season", left: "3d 15h 05m 20s"},
        ])
    })

    it("ends every day at the time of day the last one ends", () => {
        expect(today(lastDayEndsAt - 3 * DAY - (DAY - SECOND)).left).toBe("23h 59m 59s")
        expect(today(lastDayEndsAt - 3 * DAY - SECOND).left).toBe("00m 01s")
        expect(today(lastDayEndsAt - 3 * DAY).left).toBe("1d 00h 00m 00s")
        expect(today(lastDayEndsAt - 3 * DAY + SECOND).left).toBe("23h 59m 59s")
    })

    it("counts the season alone from its last day's end to its end", () => {
        expect(countdownsOf(season, lastDayEndsAt - SECOND).map((c) => c.label)).toEqual(["Today ends in", "Season ends in"])
        expect(countdownsOf(season, lastDayEndsAt)).toEqual([{label: "Season ends in", tag: "Season", left: "2h 00m 00s"}])
        expect(countdownsOf(season, endsAt - (12 * MINUTE + 5 * SECOND))).toEqual([
            {label: "Season ends in", tag: "Season", left: "12m 05s"},
        ])
    })

    it("counts a last day that ends with the season once", () => {
        const atACutoff = {...season, lastDayEndsAt: endsAt}

        expect(countdownsOf(atACutoff, endsAt - 3 * HOUR)).toEqual([{label: "Season ends in", tag: "Season", left: "3h 00m 00s"}])
        expect(countdownsOf(atACutoff, endsAt - DAY - HOUR).map((c) => c.label)).toEqual(["Today ends in", "Season ends in"])
    })

    it("rounds a part second up, so it reads 00m 01s until the end", () => {
        expect(countdownsOf(season, endsAt - 300)[0].left).toBe("00m 01s")
    })

    it("is empty once the season is over", () => {
        expect(countdownsOf(season, endsAt)).toEqual([])
        expect(countdownsOf(season, endsAt + DAY)).toEqual([])
    })
})

describe("endClock", () => {
    it("counts down to the end of the season, unit by unit", () => {
        expect(endClock(season, endsAt - (25 * DAY + 3 * HOUR + 7 * MINUTE + 9 * SECOND))).toEqual({
            left: "25d 03h 07m 09s",
            countdown: {days: 25, hours: 3, minutes: 7, seconds: 9},
        })
        expect(endClock(season, endsAt - 300)?.countdown).toEqual({days: 0, hours: 0, minutes: 0, seconds: 1})
    })

    it("is gone once the season is over", () => {
        expect(endClock(season, endsAt)).toBeUndefined()
    })
})

describe("endsLine", () => {
    it("says when the season ends in the player's own time", () => {
        expect(endsLine(season, "Europe/Paris")).toBe("Sun 1 Nov · 00:00")
        expect(endsLine(season, "UTC")).toBe("Sat 31 Oct · 23:00")
        expect(endsLine(season, "America/New_York")).toBe("Sat 31 Oct · 19:00")
    })
})
