// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {Finale} from "../../domain/battle.ts"
import {TITLE_REVEAL} from "../../domain/titleReveal.ts"
import FinaleTeaser from "./FinaleTeaser.tsx"

const HOUR = 60 * 60 * 1000
const now = Date.UTC(2026, 9, 30, 18)
const finale: Finale = {
    roundId: "season-0/finale", startsAt: now + 27 * HOUR, endsAt: now + 29 * HOUR, times: 3, description: "A bonus box every minute.", live: false,
}

beforeEach(() => {
    vi.useFakeTimers({now, shouldAdvanceTime: true})
})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
})

describe("FinaleTeaser", () => {
    it("counts down to the Final Battle and says what it is", () => {
        const play = vi.fn()
        render(<FinaleTeaser finale={finale} play={play} onClose={vi.fn()}/>)

        expect(screen.getByRole("dialog", {name: "Final Battle"})).toBeDefined()
        expect(screen.getByRole("timer").textContent).toBe("Starts in 1d 03h 00m 00s")
        expect(screen.getByText("A bonus box every minute.")).toBeDefined()
        expect(screen.getByText("Points ×3")).toBeDefined()
        expect(screen.getByText("The last word")).toBeDefined()
        expect(play).toHaveBeenCalledWith("title")
    })

    it("closes on its button once it settled", async () => {
        const onClose = vi.fn()
        render(<FinaleTeaser finale={finale} play={vi.fn()} onClose={onClose}/>)
        expect(screen.queryByRole("button", {name: "To the globe"})).toBeNull()

        act(() => vi.advanceTimersByTime(TITLE_REVEAL.ready * 1000))
        await userEvent.setup({advanceTimers: vi.advanceTimersByTime}).click(screen.getByRole("button", {name: "To the globe"}))
        expect(onClose).toHaveBeenCalledTimes(1)
    })
})
