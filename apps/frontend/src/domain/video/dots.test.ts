import {describe, expect, it} from "vitest"
import {dayOf, dotsAfter, dotsTapped, NO_DOTS, parseDots} from "./dots.ts"

describe("the video's red dots", () => {
    it("light at the first visit of the day when the flag gained or lost a tenth of a country", () => {
        expect(dotsAfter(NO_DOTS, "visit", "2026-11-12", 0.1)).toEqual({on: true, lit: "2026-11-12", visited: "2026-11-12"})
    })

    it("stay out on a smaller change, and the visit is spent", () => {
        const after = dotsAfter(NO_DOTS, "visit", "2026-11-12", 0.09)

        expect(after).toEqual({on: false, visited: "2026-11-12"})
        expect(dotsAfter(after, "visit", "2026-11-12", 0.5)).toBe(after)
    })

    it("light at the first moment of the day when the first visit did not", () => {
        const visited = dotsAfter(NO_DOTS, "visit", "2026-11-12", 0)
        const moment = dotsAfter(visited, "moment", "2026-11-12", 0.3)

        expect(moment).toEqual({on: true, lit: "2026-11-12", visited: "2026-11-12", moment: "2026-11-12"})
        expect(dotsAfter({...moment, on: false}, "moment", "2026-11-12", 0.3)).toEqual({...moment, on: false})
    })

    it("light at most once a day, however often they are put out", () => {
        const lit = dotsAfter(NO_DOTS, "visit", "2026-11-12", 0.4)
        const tapped = dotsTapped(lit)

        expect(tapped.on).toBe(false)
        expect(dotsAfter(tapped, "moment", "2026-11-12", 0.4).on).toBe(false)
        expect(dotsAfter(tapped, "visit", "2026-11-13", 0.4)).toMatchObject({on: true, lit: "2026-11-13"})
    })

    it("go out only when tapped, the next day too", () => {
        const lit = dotsAfter(NO_DOTS, "visit", "2026-11-12", 0.4)

        expect(dotsAfter(lit, "visit", "2026-11-13", 0).on).toBe(true)
        expect(dotsTapped(NO_DOTS)).toBe(NO_DOTS)
    })

    it("read back what was kept, and nothing from a broken entry", () => {
        expect(parseDots(JSON.stringify({on: true, lit: "2026-11-12", visited: 3}))).toEqual({on: true, lit: "2026-11-12"})
        expect(parseDots("{")).toEqual(NO_DOTS)
        expect(parseDots(null)).toEqual(NO_DOTS)
    })

    it("count the player's own day", () => {
        expect(dayOf(new Date(2026, 10, 2, 23, 59).getTime())).toBe("2026-11-02")
    })
})
