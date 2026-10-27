// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import home from "../../../index.html?raw"
import {finaleWindow} from "../../domain/seasonClock.ts"
import {showFinale} from "./finale.ts"

const SECOND = 1000
const MINUTE = 60 * SECOND
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR

const endsAt = Date.UTC(2026, 9, 31, 23)
const season = {number: 1, finaleStartsAt: endsAt - 2 * HOUR, endsAt}

describe("the Final Battle on the home page", () => {
    let page: Document
    let stop: () => void = () => {}

    const shown = () => [...page.querySelectorAll<HTMLElement>("[data-finale]")]
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

    it("shows the season, the battle's day and hours, and the time left before it starts", () => {
        vi.useFakeTimers({now: season.finaleStartsAt - (25 * DAY + 3 * HOUR + 7 * MINUTE + 9 * SECOND)})
        stop = showFinale(page, season)

        const when = finaleWindow(season)
        expect(shown().every((element) => !element.hidden && !element.classList.contains("finale-live"))).toBe(true)
        expect(text("season-number")).toBe("1")
        expect(text("finale-date")).toBe(when.day)
        expect(text("finale-time")).toBe(`${when.from}–${when.to}`)
        expect(text("finale-left")).toBe("25d 03h 07m 09s")
        expect(countdown()).toEqual(["25", "03", "07", "09"])

        vi.advanceTimersByTime(SECOND)
        expect(countdown()).toEqual(["25", "03", "07", "08"])
    })

    it("goes live and counts to the end once the battle starts", () => {
        vi.useFakeTimers({now: season.finaleStartsAt - SECOND})
        stop = showFinale(page, season)

        vi.advanceTimersByTime(SECOND)
        expect(shown().every((element) => element.classList.contains("finale-live"))).toBe(true)
        expect(text("finale-left")).toBe("2h 00m 00s")
        expect(countdown()).toEqual(["00", "02", "00", "00"])
    })

    it("hides and stops ticking when the season ends", () => {
        vi.useFakeTimers({now: endsAt - SECOND})
        stop = showFinale(page, season)

        vi.advanceTimersByTime(SECOND)
        expect(shown().every((element) => element.hidden)).toBe(true)
        expect(vi.getTimerCount()).toBe(0)
    })

    it("shows nothing for a season already over", () => {
        vi.useFakeTimers({now: endsAt})
        stop = showFinale(page, season)

        expect(shown().every((element) => element.hidden)).toBe(true)
        expect(vi.getTimerCount()).toBe(0)
    })
})
