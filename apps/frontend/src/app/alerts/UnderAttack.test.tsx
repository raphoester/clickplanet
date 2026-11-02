// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, fireEvent, render, screen} from "@testing-library/react"
import UnderAttack, {UNDER_ATTACK_MS} from "./UnderAttack.tsx"

afterEach(() => {
    cleanup()
    vi.useRealTimers()
})

describe("UnderAttack", () => {
    it("says the flag is under attack and leaves for Discord when pressed", () => {
        const leave = vi.fn(async () => undefined)
        render(<UnderAttack flag="fr" leave={leave} onDone={vi.fn()}/>)

        expect(screen.getByRole("status").textContent).toContain("France is under attack")
        fireEvent.click(screen.getByRole("button", {name: "Get pinged on Discord"}))

        expect(leave).toHaveBeenCalledTimes(1)
    })

    it("goes away on its own", () => {
        vi.useFakeTimers()
        const onDone = vi.fn()
        render(<UnderAttack flag="fr" leave={vi.fn(async () => undefined)} onDone={onDone}/>)

        act(() => void vi.advanceTimersByTime(UNDER_ATTACK_MS))

        expect(onDone).toHaveBeenCalledTimes(1)
    })
})
