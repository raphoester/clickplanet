// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import SeasonCard from "./SeasonCard.tsx"
import {finaleWindow} from "../../domain/seasonClock.ts"

const HOUR = 60 * 60 * 1000

const endsAt = Date.UTC(2026, 9, 31, 23)
const season = {number: 0, finaleStartsAt: endsAt - 2 * HOUR, endsAt}
const finale = finaleWindow(season)

afterEach(() => {
    cleanup()
    vi.useRealTimers()
})

describe("SeasonCard", () => {
    it("names the season, counts down to its end and offers the finale to a calendar", () => {
        vi.useFakeTimers({now: endsAt - 27 * 24 * HOUR - 14 * HOUR})
        render(<SeasonCard season={season} titleId="title"/>)

        expect(screen.getByRole("heading").textContent).toBe("Season 0")
        expect(screen.getByRole("timer").textContent).toBe("Ends in27d 14h")
        expect(screen.getByText(`${finale.day} · ${finale.from}–${finale.to}`)).toBeDefined()
        expect(screen.getByRole("button", {name: "Add to calendar"})).toBeDefined()
    })

    it("says the finale is on, with no calendar to add it to", () => {
        vi.useFakeTimers({now: endsAt - HOUR})
        const {container} = render(<SeasonCard season={season} titleId="title"/>)

        expect(screen.getByText(`Now · until ${finale.to}`)).toBeDefined()
        expect(screen.queryByRole("button", {name: "Add to calendar"})).toBeNull()
        expect(container.querySelector(".season-card-live")).not.toBeNull()
    })

    it("is gone once the season is over", () => {
        vi.useFakeTimers({now: endsAt})
        const {container} = render(<SeasonCard season={season} titleId="title"/>)

        expect(container.innerHTML).toBe("")
    })
})
