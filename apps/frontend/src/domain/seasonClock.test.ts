import {describe, expect, it} from "vitest"
import {finaleWindow, seasonClock} from "./seasonClock.ts"

const SECOND = 1000
const MINUTE = 60 * SECOND
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR

const endsAt = Date.UTC(2026, 9, 31, 23)
const season = {number: 0, finaleStartsAt: endsAt - 2 * HOUR, endsAt}
const before = {...season, finaleStartsAt: endsAt}
const left = (ms: number) => seasonClock(season, endsAt - ms)

describe("seasonClock", () => {
    it("counts days and hours while more than a day is left", () => {
        expect(left(27 * DAY + 14 * HOUR + 30 * MINUTE)).toEqual({finale: false, left: "27d 14h"})
        expect(left(DAY + 4 * HOUR)?.left).toBe("1d 04h")
        expect(left(DAY)?.left).toBe("1d 00h")
    })

    it("counts hours and minutes once less than a day is left", () => {
        expect(left(DAY - SECOND)?.left).toBe("23h 59m")
        expect(left(13 * HOUR + 5 * MINUTE + 20 * SECOND)).toEqual({finale: false, left: "13h 05m"})
        expect(left(2 * HOUR + SECOND)?.left).toBe("2h 00m")
    })

    it("counts minutes and seconds once less than an hour is left", () => {
        expect(seasonClock(before, endsAt - (52 * MINUTE + 10 * SECOND))?.left).toBe("52:10")
        expect(seasonClock(before, endsAt - (5 * MINUTE + 9 * SECOND))?.left).toBe("05:09")
    })

    it("rounds a part second up, so it reads 00:01 until the end", () => {
        expect(seasonClock(before, endsAt - 300)?.left).toBe("00:01")
        expect(seasonClock(before, endsAt - HOUR + 300)?.left).toBe("1h 00m")
    })

    it("says when the finale runs", () => {
        expect(left(2 * HOUR + SECOND)?.finale).toBe(false)
        expect(left(2 * HOUR)).toEqual({finale: true, left: "2h 00m"})
        expect(left(HOUR + 12 * MINUTE)).toEqual({finale: true, left: "1h 12m"})
        expect(left(52 * MINUTE + 10 * SECOND)).toEqual({finale: true, left: "52:10"})
    })

    it("is gone once the season is over", () => {
        expect(left(0)).toBeUndefined()
        expect(left(-SECOND)).toBeUndefined()
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
