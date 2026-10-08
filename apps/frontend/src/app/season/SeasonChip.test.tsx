// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, fireEvent, render, screen} from "@testing-library/react"
import SeasonChip, {SeasonDetails} from "./SeasonChip.tsx"
import {TURN_MS} from "./useRotation.ts"
import {STATUS_BOTTOM} from "../hud/StatusBar.tsx"
import {finaleWindow} from "../../domain/seasonClock.ts"

const SECOND = 1000
const MINUTE = 60 * SECOND
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR

const endsAt = Date.UTC(2026, 9, 31, 23)
const season = {number: 0, finaleStartsAt: endsAt - 2 * HOUR, endsAt}
const finale = finaleWindow(season)
const finaleLine = () => screen.queryByText(`${finale.day} · ${finale.from}–${finale.to}`)

const shownFace = () => [...document.querySelectorAll(".season-chip-face:not(.season-chip-face--away) > *")]
    .map((part) => part.textContent)
const turn = () => act(() => vi.advanceTimersByTime(TURN_MS))

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
        expect(shownFace()).toEqual(["Today ends in", "12h 00m 00s"])
        expect(finaleLine()).toBeNull()
    })

    it("ticks down every second", () => {
        vi.useFakeTimers({now: season.finaleStartsAt - 3 * DAY - (2 * HOUR + 52 * MINUTE + 10 * SECOND)})
        render(<SeasonChip season={season} compact={false} open={false} onToggle={vi.fn()}/>)
        expect(shownFace()).toEqual(["Today ends in", "2h 52m 10s"])

        act(() => vi.advanceTimersByTime(SECOND))
        expect(shownFace()).toEqual(["Today ends in", "2h 52m 09s"])
    })

    it("turns to the Final Battle's countdown, then back to today's", () => {
        vi.useFakeTimers({now: endsAt - 27 * DAY - 14 * HOUR})
        render(<SeasonChip season={season} compact={false} open={false} onToggle={vi.fn()}/>)

        act(() => vi.advanceTimersByTime(TURN_MS - SECOND))
        expect(shownFace()).toEqual(["Today ends in", "11h 59m 55s"])

        act(() => vi.advanceTimersByTime(SECOND))
        expect(shownFace()).toEqual(["Final Battle in", "27d 11h 59m 54s"])

        turn()
        expect(shownFace()).toEqual(["Today ends in", "11h 59m 48s"])
    })

    it("names both countdowns, and no live region, so a screen reader needs no turn", () => {
        vi.useFakeTimers({now: endsAt - 27 * DAY - 14 * HOUR})
        const {container} = render(<SeasonChip season={season} compact={false} open={false} onToggle={vi.fn()}/>)

        const name = "Today ends in 12h 00m 00s, Final Battle in 27d 12h 00m 00s"
        expect(screen.getByRole("button", {name})).toBeDefined()
        turn()
        expect(screen.getByRole("button", {name: "Today ends in 11h 59m 54s, Final Battle in 27d 11h 59m 54s"})).toBeDefined()

        expect(screen.queryByRole("timer")).toBeNull()
        expect(container.querySelector("[aria-live]")).toBeNull()
    })

    it("holds still while the mouse is over it", () => {
        vi.useFakeTimers({now: endsAt - 27 * DAY - 14 * HOUR})
        render(<SeasonChip season={season} compact={false} open={false} onToggle={vi.fn()}/>)

        fireEvent.mouseEnter(screen.getByRole("region", {name: "Season 0"}))
        turn()
        turn()
        expect(shownFace()[0]).toBe("Today ends in")

        fireEvent.mouseLeave(screen.getByRole("region", {name: "Season 0"}))
        turn()
        expect(shownFace()[0]).toBe("Final Battle in")
    })

    it("holds still while it has focus", () => {
        vi.useFakeTimers({now: endsAt - 27 * DAY - 14 * HOUR})
        render(<SeasonChip season={season} compact={false} open={false} onToggle={vi.fn()}/>)
        const chip = screen.getByRole("button", {name: /Today ends in/})

        act(() => chip.focus())
        turn()
        expect(shownFace()[0]).toBe("Today ends in")

        act(() => chip.blur())
        turn()
        expect(shownFace()[0]).toBe("Final Battle in")
    })

    it("counts the last day once, and does not turn", () => {
        vi.useFakeTimers({now: season.finaleStartsAt - 3 * HOUR})
        render(<SeasonChip season={season} compact={false} open={false} onToggle={vi.fn()}/>)

        expect(screen.getByRole("button", {name: "Final Battle in 3h 00m 00s"})).toBeDefined()
        expect(shownFace()).toEqual(["Final Battle in", "3h 00m 00s"])

        turn()
        turn()
        expect(shownFace()).toEqual(["Final Battle in", "2h 59m 48s"])
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

    it("names the finale while it runs, does not turn, and opens on what it gives", () => {
        vi.useFakeTimers({now: endsAt - HOUR - 12 * MINUTE})
        const onToggle = vi.fn()
        const {rerender} = render(<SeasonChip season={season} compact={false} open={false} onToggle={onToggle}/>)

        expect(screen.getByRole("region", {name: "Final Battle"})).toBeDefined()
        expect(shownFace()).toEqual(["Final Battle ends in", "1h 12m 00s"])
        turn()
        expect(shownFace()).toEqual(["Final Battle ends in", "1h 11m 54s"])

        fireEvent.click(screen.getByRole("button", {name: "Final Battle ends in 1h 11m 54s"}))
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
        it("is a chip with a tag and the two largest units, named in full", () => {
            vi.useFakeTimers({now: endsAt - 27 * DAY - 14 * HOUR - 5 * MINUTE})
            render(<SeasonChip season={season} compact open={false} onToggle={vi.fn()}/>)

            const chip = screen.getByRole("button", {name: "Today ends in 12h 05m 00s, Final Battle in 27d 12h 05m 00s"})
            expect(chip.getAttribute("aria-expanded")).toBe("false")
            expect(shownFace()).toEqual(["Today", "12h 05m"])

            turn()
            expect(shownFace()).toEqual(["Final", "27d 12h"])
        })

        it("holds still while the mouse is over it", () => {
            vi.useFakeTimers({now: endsAt - 27 * DAY - 14 * HOUR})
            render(<SeasonChip season={season} compact open={false} onToggle={vi.fn()}/>)

            fireEvent.mouseEnter(screen.getByRole("button"))
            turn()
            expect(shownFace()[0]).toBe("Today")
        })

        it("has no tag with one countdown", () => {
            vi.useFakeTimers({now: season.finaleStartsAt - 3 * HOUR})
            render(<SeasonChip season={season} compact open={false} onToggle={vi.fn()}/>)

            expect(screen.getByRole("button", {name: "Final Battle in 3h 00m 00s"})).toBeDefined()
            expect(shownFace()).toEqual(["3h 00m"])
        })

        it("leaves the moments to the status bar", () => {
            render(<SeasonChip season={{...season, endsAt: Date.now() + DAY}} compact open={false} onToggle={vi.fn()}/>)
            expect(document.documentElement.style.getPropertyValue(STATUS_BOTTOM)).toBe("")
        })
    })
})

describe("SeasonDetails", () => {
    it("shows both countdowns at once, and the finale's day and hours", () => {
        vi.useFakeTimers({now: endsAt - 27 * DAY})
        render(<SeasonDetails season={season}/>)

        expect(screen.getByRole("timer", {name: "Today ends in"}).textContent).toBe("22h 00m 00s")
        expect(screen.getByRole("timer", {name: "Final Battle in"}).textContent).toBe("26d 22h 00m 00s")
        expect(finaleLine()).not.toBeNull()

        turn()
        expect(screen.getByRole("timer", {name: "Today ends in"}).textContent).toBe("21h 59m 54s")
        expect(screen.getByRole("timer", {name: "Final Battle in"}).textContent).toBe("26d 21h 59m 54s")
    })

    it("says what the finale gives while it runs", () => {
        vi.useFakeTimers({now: endsAt - HOUR})
        render(<SeasonDetails season={season}/>)

        expect(screen.getAllByRole("timer")).toHaveLength(1)
        expect(screen.getByRole("timer", {name: "Final Battle ends in"}).textContent).toBe("1h 00m 00s")
        expect(screen.getByText("Power-ups for all")).toBeDefined()
        expect(finaleLine()).toBeNull()
    })
})
