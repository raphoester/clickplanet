// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, fireEvent, render, screen} from "@testing-library/react"
import {battleEndsLine} from "../../domain/battle.ts"
import BattleChip, {BattleDetails} from "./BattleChip.tsx"

const SECOND = 1000
const MINUTE = 60 * SECOND
const HOUR = 60 * MINUTE

const now = Date.UTC(2026, 9, 31, 21, 30)
const battle = {entryId: "entry-final", since: now - 30 * MINUTE, endsAt: now + HOUR + 23 * MINUTE + 5 * SECOND, description: "Triple points."}
const season = {number: 0, startsAt: Date.UTC(2026, 9, 3), endsAt: Date.UTC(2026, 11, 5, 23)}

afterEach(() => {
    cleanup()
    vi.useRealTimers()
})

describe("BattleChip", () => {
    it("counts the battle down, to the second", () => {
        vi.useFakeTimers({now})
        render(<BattleChip battle={battle} compact={false} open={false} onToggle={vi.fn()}/>)

        expect(screen.getByRole("region", {name: "Battle"})).toBeDefined()
        expect(screen.getByRole("button", {name: "Battle ends in 1h 23m 05s"})).toBeDefined()

        act(() => {
            vi.advanceTimersByTime(5 * SECOND)
        })
        expect(screen.getByRole("button", {name: "Battle ends in 1h 23m 00s"})).toBeDefined()
    })

    it("tells the battle's facts on hover, and on a press", () => {
        vi.useFakeTimers({now})
        const onToggle = vi.fn()
        render(<BattleChip battle={battle} season={season} compact={false} open={false} onToggle={onToggle}/>)
        expect(screen.queryByText("Triple points.")).toBeNull()

        fireEvent.pointerEnter(screen.getByRole("region", {name: "Battle"}))
        expect(screen.getByText("Triple points.")).toBeDefined()
        expect(screen.getByText(battleEndsLine(battle, now))).toBeDefined()
        expect(screen.getByText("Season ends in")).toBeDefined()

        fireEvent.pointerLeave(screen.getByRole("region", {name: "Battle"}))
        expect(screen.queryByText("Triple points.")).toBeNull()

        fireEvent.click(screen.getByRole("button", {name: /Battle ends in/}))
        expect(onToggle).toHaveBeenCalledTimes(1)
    })

    it("holds its facts open once pressed", () => {
        vi.useFakeTimers({now})
        render(<BattleChip battle={battle} compact={false} open={true} onToggle={vi.fn()}/>)

        expect(screen.getByText("Triple points.")).toBeDefined()
        expect(screen.getByRole("button", {name: /Battle ends in/}).getAttribute("aria-expanded")).toBe("true")
    })

    it("is a tagged pill on a phone, the two largest units alone", () => {
        vi.useFakeTimers({now})
        render(<BattleChip battle={battle} compact open={false} onToggle={vi.fn()}/>)

        const chip = screen.getByRole("button", {name: "Battle ends in 1h 23m 05s"})
        expect(chip.textContent).toBe("Battle1h 23m")
    })

    it("has a sheet with the clock and the facts", () => {
        vi.useFakeTimers({now})
        render(<BattleDetails battle={battle}/>)

        expect(screen.getByRole("timer", {name: "Battle ends in"}).textContent).toBe("1h 23m 05s")
        expect(screen.getByText("Triple points.")).toBeDefined()
    })
})
