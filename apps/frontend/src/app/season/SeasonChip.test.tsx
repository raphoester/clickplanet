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
const calendar = () => screen.queryByRole("button", {name: "Add to calendar"})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
    vi.restoreAllMocks()
})

describe("SeasonChip", () => {
    it("names the season and counts down to its end", () => {
        vi.useFakeTimers({now: endsAt - 27 * DAY - 14 * HOUR})
        render(<SeasonChip season={season} compact={false} open={false} onToggle={vi.fn()}/>)

        expect(screen.getByRole("region", {name: "Season 0"})).toBeDefined()
        expect(screen.getByText("Season 0 ends in")).toBeDefined()
        expect(screen.getByRole("timer").textContent).toBe("27d 14h 00m 00s")
        expect(calendar()).toBeNull()
    })

    it("ticks down every second", () => {
        vi.useFakeTimers({now: endsAt - 2 * HOUR - 52 * MINUTE - 10_000})
        render(<SeasonChip season={{...season, finaleStartsAt: endsAt}} compact={false} open={false} onToggle={vi.fn()}/>)
        expect(screen.getByRole("timer").textContent).toBe("2h 52m 10s")

        act(() => vi.advanceTimersByTime(1000))
        expect(screen.getByRole("timer").textContent).toBe("2h 52m 09s")
    })

    it("opens on the finale's day and hours, with a calendar to add it to", () => {
        vi.useFakeTimers({now: endsAt - 27 * DAY})
        const onToggle = vi.fn()
        const {rerender} = render(<SeasonChip season={season} compact={false} open={false} onToggle={onToggle}/>)

        fireEvent.click(screen.getByRole("button", {name: /Season 0 ends in/}))
        expect(onToggle).toHaveBeenCalledTimes(1)

        rerender(<SeasonChip season={season} compact={false} open onToggle={onToggle}/>)
        expect(screen.getByRole("button", {name: /Season 0 ends in/}).getAttribute("aria-expanded")).toBe("true")
        expect(screen.getByText(`${finale.day} · ${finale.from}–${finale.to}`)).toBeDefined()
        expect(calendar()).not.toBeNull()

        fireEvent.keyDown(document, {key: "Escape"})
        expect(onToggle).toHaveBeenCalledTimes(2)
    })

    it("closes the calendar list on Escape, and leaves the chip open", () => {
        vi.useFakeTimers({now: endsAt - 27 * DAY})
        const onToggle = vi.fn()
        render(<SeasonChip season={season} compact={false} open onToggle={onToggle}/>)

        fireEvent.click(calendar()!)
        expect(screen.getByRole("link", {name: "Google Calendar"})).toBeDefined()

        fireEvent.keyDown(document, {key: "Escape"})
        expect(screen.queryByRole("link", {name: "Google Calendar"})).toBeNull()
        expect(onToggle).not.toHaveBeenCalled()
    })

    it("names the finale while it runs, with nothing to open", () => {
        vi.useFakeTimers({now: endsAt - HOUR - 12 * MINUTE})
        render(<SeasonChip season={season} compact={false} open onToggle={vi.fn()}/>)

        expect(screen.getByRole("region", {name: "Final Battle"})).toBeDefined()
        expect(screen.getByText("Final Battle ends in")).toBeDefined()
        expect(screen.getByRole("timer").textContent).toBe("1h 12m 00s")
        expect(screen.getByRole("button")).toHaveProperty("disabled", true)
        expect(calendar()).toBeNull()
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

            const chip = screen.getByRole("button", {name: "Season 0 ends in 27d 14h 05m 00s"})
            expect(chip.textContent).toBe("27d 14h")
            expect(chip.getAttribute("aria-expanded")).toBe("false")
        })

        it("leaves the moments to the status bar", () => {
            render(<SeasonChip season={{...season, endsAt: Date.now() + DAY}} compact open={false} onToggle={vi.fn()}/>)
            expect(document.documentElement.style.getPropertyValue(STATUS_BOTTOM)).toBe("")
        })
    })
})

describe("SeasonDetails", () => {
    it("counts down and offers the finale to a calendar", () => {
        vi.useFakeTimers({now: endsAt - 27 * DAY})
        render(<SeasonDetails season={season}/>)

        expect(screen.getByRole("timer").textContent).toBe("27d 00h 00m 00s")
        expect(calendar()).not.toBeNull()
    })
})
