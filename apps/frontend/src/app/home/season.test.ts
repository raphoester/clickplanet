// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import home from "../../../index.html?raw"
import {endsLine} from "../../domain/seasonClock.ts"
import {showSeason} from "./season.ts"

const SECOND = 1000
const MINUTE = 60 * SECOND
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR

const endsAt = Date.UTC(2026, 9, 31, 23)
const season = {number: 1, lastDayEndsAt: endsAt - 2 * HOUR, endsAt}

describe("the season on the home page", () => {
    let page: Document
    let stop: () => void = () => {}

    const shown = () => [...page.querySelectorAll<HTMLElement>("[data-season]")]
    const text = (field: string) => page.querySelector(`[data-${field}]`)?.textContent
    const countdown = () => ["days", "hours", "minutes", "seconds"].map((unit) => text(`countdown-${unit}`))

    beforeEach(() => {
        page = new DOMParser().parseFromString(home, "text/html")
    })

    afterEach(() => {
        stop()
        vi.useRealTimers()
    })

    it("stays hidden until the season is known", () => {
        expect(shown()).toHaveLength(2)
        expect(shown().every((element) => element.hidden)).toBe(true)
    })

    it("shows the season, when it ends, and the time left in it", () => {
        vi.useFakeTimers({now: endsAt - (25 * DAY + 3 * HOUR + 7 * MINUTE + 9 * SECOND)})
        stop = showSeason(page, season)

        expect(shown().every((element) => !element.hidden)).toBe(true)
        expect(text("season-number")).toBe("1")
        expect(text("season-ends")).toBe(endsLine(season))
        expect(text("season-left")).toBe("25d 03h 07m 09s")
        expect(countdown()).toEqual(["25", "03", "07", "09"])

        vi.advanceTimersByTime(SECOND)
        expect(countdown()).toEqual(["25", "03", "07", "08"])
    })

    it("hides and stops ticking when the season ends", () => {
        vi.useFakeTimers({now: endsAt - SECOND})
        stop = showSeason(page, season)

        vi.advanceTimersByTime(SECOND)
        expect(shown().every((element) => element.hidden)).toBe(true)
        expect(vi.getTimerCount()).toBe(0)
    })

    it("shows nothing for a season already over", () => {
        vi.useFakeTimers({now: endsAt})
        stop = showSeason(page, season)

        expect(shown().every((element) => element.hidden)).toBe(true)
        expect(vi.getTimerCount()).toBe(0)
    })
})
