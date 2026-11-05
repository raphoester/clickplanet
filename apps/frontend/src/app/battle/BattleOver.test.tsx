// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import {BATTLE_OVER_MS} from "../../domain/battle.ts"
import BattleOver from "./BattleOver.tsx"

beforeEach(() => {
    vi.useFakeTimers()
})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
})

describe("BattleOver", () => {
    it("says the battle is over for a few seconds", () => {
        const onDone = vi.fn()
        render(<BattleOver awaiting={false} onDone={onDone}/>)

        expect(screen.getByRole("status").textContent).toBe("The battle is over")
        act(() => vi.advanceTimersByTime(BATTLE_OVER_MS))
        expect(onDone).toHaveBeenCalledOnce()
    })

    it("says the results are coming until they come, then goes", () => {
        const onDone = vi.fn()
        const {rerender} = render(<BattleOver awaiting onDone={onDone}/>)

        expect(screen.getByText("The results are coming")).toBeDefined()
        act(() => vi.advanceTimersByTime(3 * BATTLE_OVER_MS))
        expect(onDone).not.toHaveBeenCalled()

        rerender(<BattleOver awaiting={false} onDone={onDone}/>)
        act(() => vi.advanceTimersByTime(0))
        expect(onDone).toHaveBeenCalledOnce()
    })
})
