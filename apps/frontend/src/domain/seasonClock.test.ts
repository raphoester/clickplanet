import {describe, expect, it} from "vitest"
import {countdownsOf, endClock, endsLine} from "./seasonClock.ts"

const SECOND = 1000
const MINUTE = 60 * SECOND
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR

const endsAt = Date.UTC(2026, 9, 31, 23)
const season = {number: 0, startsAt: Date.UTC(2026, 9, 3), endsAt, bookId: "season-0"}
const lastDayEndsAt = endsAt - 2 * HOUR

const day = {number: 21, startsAt: lastDayEndsAt - 4 * DAY, endsAt: lastDayEndsAt - 3 * DAY, roundId: "season-0/day-21"}

describe("countdownsOf", () => {
    it("counts down to today's end and to the season's", () => {
        expect(countdownsOf(season, day.endsAt - (13 * HOUR + 5 * MINUTE + 20 * SECOND), undefined, day.endsAt)).toEqual([
            {label: "Today ends in", tag: "Today", left: "13h 05m 20s"},
            {label: "Season ends in", tag: "Season", left: "3d 15h 05m 20s"},
        ])
    })

    it("counts the season alone with no day running", () => {
        expect(countdownsOf(season, lastDayEndsAt)).toEqual([{label: "Season ends in", tag: "Season", left: "2h 00m 00s"}])
        expect(countdownsOf(season, lastDayEndsAt, undefined, lastDayEndsAt).map((c) => c.label)).toEqual(["Season ends in"])
        expect(countdownsOf(season, endsAt - (12 * MINUTE + 5 * SECOND))).toEqual([
            {label: "Season ends in", tag: "Season", left: "12m 05s"},
        ])
    })

    it("counts a day that ends with the season once", () => {
        expect(countdownsOf(season, endsAt - 3 * HOUR, undefined, endsAt)).toEqual([{label: "Season ends in", tag: "Season", left: "3h 00m 00s"}])
    })

    it("rounds a part second up, so it reads 00m 01s until the end", () => {
        expect(countdownsOf(season, endsAt - 300)[0].left).toBe("00m 01s")
    })

    it("is empty once the season is over", () => {
        expect(countdownsOf(season, endsAt)).toEqual([])
        expect(countdownsOf(season, endsAt + DAY)).toEqual([])
    })

    const battle = {entryId: "finale", startsAt: lastDayEndsAt, endsAt, description: "Triple points.", times: 3, endsSeason: true, roundId: "finale"}

    it("counts down to the next battle between today's end and the season's", () => {
        expect(countdownsOf(season, lastDayEndsAt - DAY - 3 * HOUR, battle, lastDayEndsAt - DAY)).toEqual([
            {label: "Today ends in", tag: "Today", left: "3h 00m 00s"},
            {label: "Next battle in", tag: "Battle", left: "1d 03h 00m 00s"},
            {label: "Season ends in", tag: "Season", left: "1d 05h 00m 00s"},
        ])
    })

    it("drops the battle once it starts", () => {
        expect(countdownsOf(season, battle.startsAt - SECOND, battle, lastDayEndsAt).map((c) => c.tag)).toEqual(["Today", "Battle", "Season"])
        expect(countdownsOf(season, battle.startsAt, battle, lastDayEndsAt).map((c) => c.tag)).toEqual(["Season"])
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
