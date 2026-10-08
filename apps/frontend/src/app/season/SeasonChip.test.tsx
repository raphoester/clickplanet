// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, fireEvent, render, screen} from "@testing-library/react"
import SeasonChip, {SeasonDetails} from "./SeasonChip.tsx"
import {STATUS_BOTTOM} from "../hud/StatusBar.tsx"
import {finaleWindow} from "../../domain/seasonClock.ts"

const MINUTE = 60 * 1000
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR

const endsAt = Date.UTC(2026, 9, 31, 23)
const season = {number: 0, finaleStartsAt: endsAt - 2 * HOUR, endsAt}
const finale = finaleWindow(season)
const finaleLine = () => screen.queryByText(`${finale.day} · ${finale.from}–${finale.to}`)

afterEach(() => {
    cleanup()
    vi.useRealTimers()
    vi.restoreAllMocks()
})

describe("SeasonChip", () => {
    it("counts down to today's cutoff, at the time of day the Final Battle starts", () => {
        vi.useFakeTimers({now: endsAt - 27 * DAY - 14 * HOUR})
        render(<SeasonChip season={season} compact={false} open={false} onToggle={vi.fn()}/>)

        expect(screen.getByRole("region", {name: "Season 0"})).toBeDefined()
        expect(screen.getByText("Today ends in")).toBeDefined()
        expect(screen.getByRole("timer").textContent).toBe("12h 00m 00s")
        expect(finaleLine()).toBeNull()
    })

    it("ticks down every second", () => {
        vi.useFakeTimers({now: endsAt - 2 * HOUR - 52 * MINUTE - 10_000})
        render(<SeasonChip season={{...season, finaleStartsAt: endsAt}} compact={false} open={false} onToggle={vi.fn()}/>)
        expect(screen.getByRole("timer").textContent).toBe("2h 52m 10s")

        act(() => vi.advanceTimersByTime(1000))
        expect(screen.getByRole("timer").textContent).toBe("2h 52m 09s")
    })

    it("opens on the finale's day and hours", () => {
        vi.useFakeTimers({now: endsAt - 27 * DAY})
        const onToggle = vi.fn()
        const {rerender} = render(<SeasonChip season={season} compact={false} open={false} onToggle={onToggle}/>)

        fireEvent.click(screen.getByRole("button", {name: /Today ends in/}))
        expect(onToggle).toHaveBeenCalledTimes(1)

        rerender(<SeasonChip season={season} compact={false} open onToggle={onToggle}/>)
        expect(screen.getByRole("button", {name: /Today ends in/}).getAttribute("aria-expanded")).toBe("true")
        expect(finaleLine()).not.toBeNull()
        expect(screen.getByText("Each day, the countries that held the most ground score points. The Final Battle scores triple.")).toBeDefined()

        fireEvent.keyDown(document, {key: "Escape"})
        expect(onToggle).toHaveBeenCalledTimes(2)
    })

    it("names the finale while it runs, and opens on what it gives", () => {
        vi.useFakeTimers({now: endsAt - HOUR - 12 * MINUTE})
        const onToggle = vi.fn()
        const {rerender} = render(<SeasonChip season={season} compact={false} open={false} onToggle={onToggle}/>)

        expect(screen.getByRole("region", {name: "Final Battle"})).toBeDefined()
        expect(screen.getByText("Final Battle ends in")).toBeDefined()
        expect(screen.getByRole("timer").textContent).toBe("1h 12m 00s")

        fireEvent.click(screen.getByRole("button", {name: /Final Battle ends in/}))
        expect(onToggle).toHaveBeenCalledTimes(1)

        rerender(<SeasonChip season={season} compact={false} open onToggle={onToggle}/>)
        expect(screen.getByText("Power-ups for all")).toBeDefined()
        expect(screen.getByText("The countries holding the most ground score triple points.")).toBeDefined()
        expect(finaleLine()).toBeNull()
    })

    it("is gone once the season is over", () => {
        vi.useFakeTimers({now: endsAt})
        const {container} = render(<SeasonChip season={season} compact={false} open={false} onToggle={vi.fn()}/>)
        expect(container.innerHTML).toBe("")
    })

    it("tells the moments at the top where it ends, so they slide under it", () => {
        vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockReturnValue(new DOMRect(0, 16, 376, 51.4))
        const bottom = () => document.documentElement.style.getPropertyValue(STATUS_BOTTOM)

        const {unmount} = render(<SeasonChip season={{...season, endsAt: Date.now() + DAY}} compact={false} open={false} onToggle={vi.fn()}/>)
        expect(bottom()).toBe("68px")

        unmount()
        expect(bottom()).toBe("")
    })

    describe("on a phone", () => {
        it("is a chip with the two largest units, named in full", () => {
            vi.useFakeTimers({now: endsAt - 27 * DAY - 14 * HOUR - 5 * MINUTE})
            render(<SeasonChip season={season} compact open={false} onToggle={vi.fn()}/>)

            const chip = screen.getByRole("button", {name: "Today ends in 12h 05m 00s"})
            expect(chip.textContent).toBe("12h 05m")
            expect(chip.getAttribute("aria-expanded")).toBe("false")
        })

        it("leaves the moments to the status bar", () => {
            render(<SeasonChip season={{...season, endsAt: Date.now() + DAY}} compact open={false} onToggle={vi.fn()}/>)
            expect(document.documentElement.style.getPropertyValue(STATUS_BOTTOM)).toBe("")
        })
    })
})

describe("SeasonDetails", () => {
    it("counts down and names the finale's day and hours", () => {
        vi.useFakeTimers({now: endsAt - 27 * DAY})
        render(<SeasonDetails season={season}/>)

        expect(screen.getByText("Today ends in")).toBeDefined()
        expect(screen.getByRole("timer").textContent).toBe("22h 00m 00s")
        expect(finaleLine()).not.toBeNull()
    })

    it("says what the finale gives while it runs", () => {
        vi.useFakeTimers({now: endsAt - HOUR})
        render(<SeasonDetails season={season}/>)

        expect(screen.getByText("Final Battle ends in")).toBeDefined()
        expect(screen.getByText("Power-ups for all")).toBeDefined()
        expect(finaleLine()).toBeNull()
    })
})
