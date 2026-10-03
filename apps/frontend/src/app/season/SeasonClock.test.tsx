// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import SeasonClock from "./SeasonClock.tsx"

const MINUTE = 60 * 1000
const HOUR = 60 * MINUTE

const endsAt = Date.UTC(2026, 9, 31, 23)
const season = {number: 0, finaleStartsAt: endsAt - 2 * HOUR, endsAt}

afterEach(() => {
    cleanup()
    vi.useRealTimers()
})

describe("SeasonClock", () => {
    it("ticks down every second, then goes at the end", () => {
        vi.useFakeTimers({now: endsAt - 52 * MINUTE - 10_000})
        render(<SeasonClock season={{...season, finaleStartsAt: endsAt}}/>)
        expect(screen.getByRole("timer").textContent).toBe("Season 0 · 52:10")

        act(() => vi.advanceTimersByTime(1000))
        expect(screen.getByRole("timer").textContent).toBe("Season 0 · 52:09")

        act(() => vi.advanceTimersByTime(52 * MINUTE + 9_000))
        expect(screen.queryByRole("timer")).toBeNull()
    })

    it("names the finale in the accent colour", () => {
        vi.useFakeTimers({now: endsAt - HOUR - 12 * MINUTE})
        render(<SeasonClock season={season}/>)

        const clock = screen.getByRole("timer")
        expect(clock.textContent).toBe("Final Assault · 1h 12m")
        expect(clock.classList.contains("season-clock-finale")).toBe(true)
    })

    it("names the season before the finale", () => {
        vi.useFakeTimers({now: endsAt - 27 * 24 * HOUR - 14 * HOUR})
        render(<SeasonClock season={season}/>)

        const clock = screen.getByRole("timer")
        expect(clock.textContent).toBe("Season 0 · 27d 14h")
        expect(clock.classList.contains("season-clock-finale")).toBe(false)
    })
})
