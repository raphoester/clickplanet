import {describe, expect, it} from "vitest"
import {finaleClock, finaleWindow, seasonClock} from "./seasonClock.ts"

const SECOND = 1000
const MINUTE = 60 * SECOND
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR

const endsAt = Date.UTC(2026, 9, 31, 23)
const season = {number: 0, finaleStartsAt: endsAt - 2 * HOUR, endsAt}
const before = {...season, finaleStartsAt: endsAt}
const left = (ms: number) => seasonClock(season, endsAt - ms)

describe("seasonClock", () => {
    const finaleStartsAt = season.finaleStartsAt
    const today = (beforeCutoff: number) => seasonClock(season, finaleStartsAt - 3 * DAY - beforeCutoff)

    it("counts down to today's cutoff, at the time of day the Final Battle starts", () => {
        expect(today(13 * HOUR + 5 * MINUTE + 20 * SECOND)).toEqual({finale: false, left: "13h 05m 20s"})
        expect(today(DAY - SECOND)?.left).toBe("23h 59m 59s")
        expect(today(SECOND)?.left).toBe("00m 01s")
    })

    it("starts the next day at the cutoff", () => {
        expect(today(0)?.left).toBe("1d 00h 00m 00s")
        expect(today(-SECOND)?.left).toBe("23h 59m 59s")
    })

    it("counts the last day down to the start of the Final Battle", () => {
        expect(left(2 * HOUR + SECOND)).toEqual({finale: false, left: "00m 01s"})
        expect(left(5 * HOUR)?.left).toBe("3h 00m 00s")
    })

    it("drops the hours once less than an hour is left", () => {
        expect(seasonClock(before, endsAt - (52 * MINUTE + 10 * SECOND))?.left).toBe("52m 10s")
        expect(seasonClock(before, endsAt - (5 * MINUTE + 9 * SECOND))?.left).toBe("05m 09s")
    })

    it("rounds a part second up, so it reads 00m 01s until the end", () => {
        expect(seasonClock(before, endsAt - 300)?.left).toBe("00m 01s")
        expect(seasonClock(before, endsAt - HOUR + 300)?.left).toBe("1h 00m 00s")
    })

    it("says when the finale runs", () => {
        expect(left(2 * HOUR + SECOND)?.finale).toBe(false)
        expect(left(2 * HOUR)).toEqual({finale: true, left: "2h 00m 00s"})
        expect(left(HOUR + 12 * MINUTE)).toEqual({finale: true, left: "1h 12m 00s"})
        expect(left(52 * MINUTE + 10 * SECOND)).toEqual({finale: true, left: "52m 10s"})
    })

    it("is gone once the season is over", () => {
        expect(left(0)).toBeUndefined()
        expect(left(-SECOND)).toBeUndefined()
    })
})

describe("finaleClock", () => {
    const finaleStartsAt = season.finaleStartsAt

    it("counts down to the start of the Final Battle, unit by unit", () => {
        expect(finaleClock(season, finaleStartsAt - (25 * DAY + 3 * HOUR + 7 * MINUTE + 9 * SECOND))).toEqual({
            live: false,
            left: "25d 03h 07m 09s",
            countdown: {days: 25, hours: 3, minutes: 7, seconds: 9},
        })
        expect(finaleClock(season, finaleStartsAt - 300)?.countdown).toEqual({days: 0, hours: 0, minutes: 0, seconds: 1})
    })

    it("counts down to the end of the season once the Final Battle runs", () => {
        expect(finaleClock(season, finaleStartsAt)).toEqual({
            live: true,
            left: "2h 00m 00s",
            countdown: {days: 0, hours: 2, minutes: 0, seconds: 0},
        })
        expect(finaleClock(season, endsAt - (12 * MINUTE + 5 * SECOND))?.left).toBe("12m 05s")
    })

    it("is gone once the season is over", () => {
        expect(finaleClock(season, endsAt)).toBeUndefined()
    })
})

describe("finaleWindow", () => {
    it("says when the finale runs in the player's own time", () => {
        expect(finaleWindow(season, "Europe/Paris")).toEqual({day: "Sat 31 Oct", from: "22:00", to: "00:00"})
        expect(finaleWindow(season, "UTC")).toEqual({day: "Sat 31 Oct", from: "21:00", to: "23:00"})
        expect(finaleWindow(season, "America/New_York")).toEqual({day: "Sat 31 Oct", from: "17:00", to: "19:00"})
        expect(finaleWindow(season, "Asia/Kolkata")).toEqual({day: "Sun 1 Nov", from: "02:30", to: "04:30"})
    })
})
