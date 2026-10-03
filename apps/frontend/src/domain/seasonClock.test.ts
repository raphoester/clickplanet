import {describe, expect, it} from "vitest"
import {seasonClock, seasonEnd} from "./seasonClock.ts"

const SECOND = 1000
const MINUTE = 60 * SECOND
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR

const endsAt = Date.UTC(2026, 9, 31, 23)
const season = {number: 0, finaleStartsAt: endsAt - 2 * HOUR, endsAt}
const left = (ms: number) => seasonClock(season, endsAt - ms)

describe("seasonClock", () => {
    it("counts days and hours while more than a day is left", () => {
        expect(left(27 * DAY + 14 * HOUR + 30 * MINUTE)).toEqual({text: "S0 · 27d 14h", finale: false})
        expect(left(DAY + 4 * HOUR)).toEqual({text: "S0 · 1d 04h", finale: false})
        expect(left(DAY)).toEqual({text: "S0 · 1d 00h", finale: false})
    })

    it("counts hours and minutes once less than a day is left", () => {
        expect(left(DAY - SECOND)).toEqual({text: "23h 59m", finale: false})
        expect(left(13 * HOUR + 5 * MINUTE + 20 * SECOND)).toEqual({text: "13h 05m", finale: false})
        expect(left(2 * HOUR + SECOND)).toEqual({text: "2h 00m", finale: false})
    })

    it("counts minutes and seconds once less than an hour is left", () => {
        const before = {...season, finaleStartsAt: endsAt}
        expect(seasonClock(before, endsAt - (52 * MINUTE + 10 * SECOND))).toEqual({text: "52:10", finale: false})
        expect(seasonClock(before, endsAt - (5 * MINUTE + 9 * SECOND))).toEqual({text: "05:09", finale: false})
    })

    it("rounds a part second up, so it reads 00:01 until the end", () => {
        const before = {...season, finaleStartsAt: endsAt}
        expect(seasonClock(before, endsAt - 300)).toEqual({text: "00:01", finale: false})
        expect(seasonClock(before, endsAt - HOUR + 300)).toEqual({text: "1h 00m", finale: false})
    })

    it("names the finale while it runs", () => {
        expect(left(2 * HOUR)).toEqual({text: "Final Assault · 2h 00m", finale: true})
        expect(left(HOUR + 12 * MINUTE)).toEqual({text: "Final Assault · 1h 12m", finale: true})
        expect(left(52 * MINUTE + 10 * SECOND)).toEqual({text: "Final Assault · 52:10", finale: true})
    })

    it("is gone once the season is over", () => {
        expect(left(0)).toBeUndefined()
        expect(left(-SECOND)).toBeUndefined()
    })

    it("names the season by its number", () => {
        expect(seasonClock({...season, number: 3}, endsAt - 3 * DAY)?.text).toBe("S3 · 3d 00h")
    })
})

describe("seasonEnd", () => {
    it("says when the season ends in the player's own time", () => {
        expect(seasonEnd(season, "Europe/Paris")).toBe("Ends Sun 1 Nov, 00:00")
        expect(seasonEnd(season, "UTC")).toBe("Ends Sat 31 Oct, 23:00")
        expect(seasonEnd(season, "America/New_York")).toBe("Ends Sat 31 Oct, 19:00")
        expect(seasonEnd(season, "Asia/Kolkata")).toBe("Ends Sun 1 Nov, 04:30")
    })
})
