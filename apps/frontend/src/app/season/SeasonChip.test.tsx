// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, fireEvent, render, screen} from "@testing-library/react"
import SeasonChip, {ChampionChip, SeasonDetails} from "./SeasonChip.tsx"
import {TURN_MS} from "./useRotation.ts"
import {STATUS_BOTTOM} from "../hud/StatusBar.tsx"

const SECOND = 1000
const MINUTE = 60 * SECOND
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR

const endsAt = Date.UTC(2026, 9, 31, 23)
const season = {number: 0, lastDayEndsAt: endsAt - 2 * HOUR, endsAt}
const battleTeased = () => screen.queryByText(/Final Battle/)

const shownFace = () => [...document.querySelectorAll(".season-chip-face:not(.season-chip-face--away) > *")]
    .map((part) => part.textContent)
const turn = () => act(() => vi.advanceTimersByTime(TURN_MS))

afterEach(() => {
    cleanup()
    vi.useRealTimers()
    vi.restoreAllMocks()
})

describe("SeasonChip", () => {
    it("counts down to today's cutoff, at the time of day the last day ends", () => {
        vi.useFakeTimers({now: endsAt - 27 * DAY - 14 * HOUR})
        render(<SeasonChip season={season} compact={false} open={false} onToggle={vi.fn()}/>)

        expect(screen.getByRole("region", {name: "Season 0"})).toBeDefined()
        expect(shownFace()).toEqual(["Today ends in", "12h 00m 00s"])
        expect(battleTeased()).toBeNull()
    })

    it("turns to the next battle's countdown between today's and the season's", () => {
        vi.useFakeTimers({now: endsAt - 27 * DAY - 14 * HOUR})
        const nextBattle = {entryId: "rush", startsAt: Date.now() + DAY, endsAt: Date.now() + DAY + 2 * HOUR, description: ""}
        render(<SeasonChip season={season} nextBattle={nextBattle} compact={false} open={false} onToggle={vi.fn()}/>)

        turn()
        expect(shownFace()).toEqual(["Next battle in", "23h 59m 54s"])
        turn()
        expect(shownFace()[0]).toBe("Season ends in")
    })

    it("ticks down every second", () => {
        vi.useFakeTimers({now: season.lastDayEndsAt - 3 * DAY - (2 * HOUR + 52 * MINUTE + 10 * SECOND)})
        render(<SeasonChip season={season} compact={false} open={false} onToggle={vi.fn()}/>)
        expect(shownFace()).toEqual(["Today ends in", "2h 52m 10s"])

        act(() => vi.advanceTimersByTime(SECOND))
        expect(shownFace()).toEqual(["Today ends in", "2h 52m 09s"])
    })

    it("turns to the season's countdown, then back to today's", () => {
        vi.useFakeTimers({now: endsAt - 27 * DAY - 14 * HOUR})
        render(<SeasonChip season={season} compact={false} open={false} onToggle={vi.fn()}/>)

        act(() => vi.advanceTimersByTime(TURN_MS - SECOND))
        expect(shownFace()).toEqual(["Today ends in", "11h 59m 55s"])

        act(() => vi.advanceTimersByTime(SECOND))
        expect(shownFace()).toEqual(["Season ends in", "27d 13h 59m 54s"])

        turn()
        expect(shownFace()).toEqual(["Today ends in", "11h 59m 48s"])
    })

    it("names both countdowns, and no live region, so a screen reader needs no turn", () => {
        vi.useFakeTimers({now: endsAt - 27 * DAY - 14 * HOUR})
        const {container} = render(<SeasonChip season={season} compact={false} open={false} onToggle={vi.fn()}/>)

        const name = "Today ends in 12h 00m 00s, Season ends in 27d 14h 00m 00s"
        expect(screen.getByRole("button", {name})).toBeDefined()
        turn()
        expect(screen.getByRole("button", {name: "Today ends in 11h 59m 54s, Season ends in 27d 13h 59m 54s"})).toBeDefined()

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
        expect(shownFace()[0]).toBe("Season ends in")
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
        expect(shownFace()[0]).toBe("Season ends in")
    })

    it("counts the season alone once its last day ended, and does not turn", () => {
        vi.useFakeTimers({now: endsAt - HOUR - 12 * MINUTE})
        render(<SeasonChip season={season} compact={false} open={false} onToggle={vi.fn()}/>)

        expect(screen.getByRole("region", {name: "Season 0"})).toBeDefined()
        expect(screen.getByRole("button", {name: "Season ends in 1h 12m 00s"})).toBeDefined()
        expect(shownFace()).toEqual(["Season ends in", "1h 12m 00s"])

        turn()
        turn()
        expect(shownFace()).toEqual(["Season ends in", "1h 11m 48s"])
    })

    it("opens on what a season is, and promises no battle", () => {
        vi.useFakeTimers({now: endsAt - 27 * DAY})
        const onToggle = vi.fn()
        const {rerender} = render(<SeasonChip season={season} compact={false} open={false} onToggle={onToggle}/>)

        fireEvent.click(screen.getByRole("button", {name: /Today ends in/}))
        expect(onToggle).toHaveBeenCalledTimes(1)

        rerender(<SeasonChip season={season} compact={false} open onToggle={onToggle}/>)
        expect(screen.getByRole("button", {name: /Today ends in/}).getAttribute("aria-expanded")).toBe("true")
        expect(screen.getByText("When a day ends, the countries holding the most ground score points.")).toBeDefined()
        expect(battleTeased()).toBeNull()

        fireEvent.keyDown(document, {key: "Escape"})
        expect(onToggle).toHaveBeenCalledTimes(2)
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

            const chip = screen.getByRole("button", {name: "Today ends in 12h 05m 00s, Season ends in 27d 14h 05m 00s"})
            expect(chip.getAttribute("aria-expanded")).toBe("false")
            expect(shownFace()).toEqual(["Today", "12h 05m"])

            turn()
            expect(shownFace()).toEqual(["Season", "27d 14h"])
        })

        it("holds still while the mouse is over it", () => {
            vi.useFakeTimers({now: endsAt - 27 * DAY - 14 * HOUR})
            render(<SeasonChip season={season} compact open={false} onToggle={vi.fn()}/>)

            fireEvent.mouseEnter(screen.getByRole("button"))
            turn()
            expect(shownFace()[0]).toBe("Today")
        })

        it("has no tag with one countdown", () => {
            vi.useFakeTimers({now: endsAt - HOUR})
            render(<SeasonChip season={season} compact open={false} onToggle={vi.fn()}/>)

            expect(screen.getByRole("button", {name: "Season ends in 1h 00m 00s"})).toBeDefined()
            expect(shownFace()).toEqual(["1h 00m"])
        })

        it("leaves the moments to the status bar", () => {
            render(<SeasonChip season={{...season, endsAt: Date.now() + DAY}} compact open={false} onToggle={vi.fn()}/>)
            expect(document.documentElement.style.getPropertyValue(STATUS_BOTTOM)).toBe("")
        })
    })
})

describe("SeasonDetails", () => {
    it("shows both countdowns at once, and promises no battle", () => {
        vi.useFakeTimers({now: endsAt - 27 * DAY})
        render(<SeasonDetails season={season}/>)

        expect(screen.getByRole("timer", {name: "Today ends in"}).textContent).toBe("22h 00m 00s")
        expect(screen.getByRole("timer", {name: "Season ends in"}).textContent).toBe("27d 00h 00m 00s")
        expect(battleTeased()).toBeNull()

        turn()
        expect(screen.getByRole("timer", {name: "Today ends in"}).textContent).toBe("21h 59m 54s")
        expect(screen.getByRole("timer", {name: "Season ends in"}).textContent).toBe("26d 23h 59m 54s")
    })

    it("counts the season alone once its last day ended", () => {
        vi.useFakeTimers({now: endsAt - HOUR})
        render(<SeasonDetails season={season}/>)

        expect(screen.getAllByRole("timer")).toHaveLength(1)
        expect(screen.getByRole("timer", {name: "Season ends in"}).textContent).toBe("1h 00m 00s")
    })
})

describe("ChampionChip", () => {
    const germany = {season: 0, countryCodes: ["de"]}

    it("shows the champion of the season that ended", () => {
        render(<ChampionChip champion={germany} compact={false}/>)

        const chip = screen.getByRole("region", {name: "Germany wins Season 0"})
        expect(chip.textContent).toContain("Season 0 champion")
        expect(chip.textContent).toContain("Germany")
        expect(chip.textContent).toContain("🏆")
    })

    it("names champions together", () => {
        render(<ChampionChip champion={{season: 0, countryCodes: ["de", "fr"]}} compact/>)

        expect(screen.getByRole("region", {name: "Germany and France win Season 0"})).toBeDefined()
    })

    it("tells the moments at the top where it ends, as the countdown did", () => {
        vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockReturnValue(new DOMRect(0, 16, 376, 51.4))
        const bottom = () => document.documentElement.style.getPropertyValue(STATUS_BOTTOM)

        const {unmount} = render(<ChampionChip champion={germany} compact={false}/>)
        expect(bottom()).toBe("68px")

        unmount()
        expect(bottom()).toBe("")
    })

    it("on a phone, is the trophy and the flag, and leaves the moments to the status bar", () => {
        render(<ChampionChip champion={germany} compact/>)

        const chip = screen.getByRole("region", {name: "Germany wins Season 0"})
        expect(chip.textContent).toBe("🏆")
        expect(chip.querySelectorAll(".country-flag")).toHaveLength(1)
        expect(document.documentElement.style.getPropertyValue(STATUS_BOTTOM)).toBe("")
    })
})
