// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from 'vitest'
import {act, cleanup, fireEvent, render, screen} from '@testing-library/react'
import StreakFlame from './StreakFlame.tsx'
import {BUBBLE_MS} from './Bubble.tsx'

afterEach(cleanup)

describe("StreakFlame", () => {
    it("draws nothing under three days", () => {
        const {container} = render(<StreakFlame days={2}/>)

        expect(container.innerHTML).toBe("")
    })

    it("says what the flame counts when it is tapped, then lets it go", () => {
        vi.useFakeTimers()
        try {
            render(<StreakFlame days={12}/>)

            fireEvent.click(screen.getByRole("button", {name: "12-day streak"}))
            expect(screen.getByRole("status").textContent).toBe("Played 12 days in a row")

            act(() => void vi.advanceTimersByTime(BUBBLE_MS))
            expect(screen.queryByRole("status")).toBeNull()
        } finally {
            vi.useRealTimers()
        }
    })

    it("says it while a mouse rests on it", () => {
        render(<StreakFlame days={5}/>)
        const flame = screen.getByRole("button", {name: "5-day streak"})

        fireEvent.pointerEnter(flame, {pointerType: "mouse"})
        expect(screen.getByRole("status").textContent).toBe("Played 5 days in a row")

        fireEvent.pointerLeave(flame, {pointerType: "mouse"})
        expect(screen.queryByRole("status")).toBeNull()
    })
})
