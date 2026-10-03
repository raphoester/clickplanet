// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import SeasonBanner, {SEASON_BANNER_BOTTOM, SEASON_BANNER_FOLDED_KEY} from "./SeasonBanner.tsx"
import {finaleWindow} from "../../domain/seasonClock.ts"

const MINUTE = 60 * 1000
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR

const endsAt = Date.UTC(2026, 9, 31, 23)
const season = {number: 0, finaleStartsAt: endsAt - 2 * HOUR, endsAt}
const finale = finaleWindow(season)
const calendar = () => screen.queryByRole("button", {name: "Add to calendar"})

beforeEach(() => window.localStorage.clear())

afterEach(() => {
    cleanup()
    vi.useRealTimers()
    vi.restoreAllMocks()
})

describe("SeasonBanner", () => {
    it("names the season, counts down to its end and offers the finale to a calendar", () => {
        vi.useFakeTimers({now: endsAt - 27 * DAY - 14 * HOUR})
        render(<SeasonBanner season={season}/>)

        expect(screen.getByRole("region", {name: "Season 0"})).toBeDefined()
        expect(screen.getByText("Season 0 ends in")).toBeDefined()
        expect(screen.getByRole("timer").textContent).toBe("27d 14h 00m 00s")
        expect(screen.getByText(`${finale.day} · ${finale.from}–${finale.to}`)).toBeDefined()
        expect(calendar()).not.toBeNull()
    })

    it("ticks down every second", () => {
        vi.useFakeTimers({now: endsAt - 2 * HOUR - 52 * MINUTE - 10_000})
        render(<SeasonBanner season={{...season, finaleStartsAt: endsAt}}/>)
        expect(screen.getByRole("timer").textContent).toBe("2h 52m 10s")

        act(() => vi.advanceTimersByTime(1000))
        expect(screen.getByRole("timer").textContent).toBe("2h 52m 09s")

        act(() => vi.advanceTimersByTime(2 * HOUR))
        expect(screen.getByRole("timer").textContent).toBe("52m 09s")
    })

    it("folds to the countdown alone, and stays folded on the next load", async () => {
        const user = userEvent.setup()
        const {unmount} = render(<SeasonBanner season={{...season, endsAt: Date.now() + DAY, finaleStartsAt: Date.now() + DAY - HOUR}}/>)

        const fold = screen.getByRole("button", {name: /Season 0 ends in/})
        expect(fold.getAttribute("aria-expanded")).toBe("true")

        await user.click(fold)
        expect(fold.getAttribute("aria-expanded")).toBe("false")
        expect(calendar()).toBeNull()
        expect(window.localStorage.getItem(SEASON_BANNER_FOLDED_KEY)).toBe("1")

        unmount()
        render(<SeasonBanner season={{...season, endsAt: Date.now() + DAY, finaleStartsAt: Date.now() + DAY - HOUR}}/>)
        expect(screen.getByRole("button", {name: /Season 0 ends in/}).getAttribute("aria-expanded")).toBe("false")
    })

    it("names the finale while it runs, with nothing left to add to a calendar", () => {
        vi.useFakeTimers({now: endsAt - HOUR - 12 * MINUTE})
        render(<SeasonBanner season={season}/>)

        expect(screen.getByRole("region", {name: "Final Battle"})).toBeDefined()
        expect(screen.getByText("Final Battle ends in")).toBeDefined()
        expect(screen.getByRole("timer").textContent).toBe("1h 12m 00s")
        expect(screen.queryByRole("button")).toBeNull()
    })

    it("is gone once the season is over", () => {
        vi.useFakeTimers({now: endsAt})
        const {container} = render(<SeasonBanner season={season}/>)

        expect(container.innerHTML).toBe("")
    })

    it("tells the notes at the top where it ends, so they slide under it", () => {
        vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockReturnValue(new DOMRect(470, 16, 330, 100.4))
        const bottom = () => document.documentElement.style.getPropertyValue(SEASON_BANNER_BOTTOM)

        const {unmount} = render(<SeasonBanner season={{...season, endsAt: Date.now() + DAY, finaleStartsAt: Date.now() + DAY - HOUR}}/>)
        expect(bottom()).toBe("117px")

        unmount()
        expect(bottom()).toBe("")
    })
})
