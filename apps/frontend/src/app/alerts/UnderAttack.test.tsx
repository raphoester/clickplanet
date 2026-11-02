// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, fireEvent, render, screen} from "@testing-library/react"
import UnderAttack, {UNDER_ATTACK_MS} from "./UnderAttack.tsx"

afterEach(() => {
    cleanup()
    vi.useRealTimers()
})

describe("UnderAttack", () => {
    it("says the flag is under attack and connects Discord when pressed", () => {
        const connect = vi.fn(async () => undefined)
        render(<UnderAttack flag="fr" connect={connect} onDone={vi.fn()}/>)

        expect(screen.getByRole("status").textContent).toContain("France is under attack")
        const ping = screen.getByRole("button", {name: "Join Discord and get pinged when France is attacked"})
        expect(ping.textContent).toBe("Get pinged")
        fireEvent.click(ping)

        expect(connect).toHaveBeenCalledTimes(1)
    })

    it("goes away on its own", () => {
        vi.useFakeTimers()
        const onDone = vi.fn()
        render(<UnderAttack flag="fr" connect={vi.fn(async () => undefined)} onDone={onDone}/>)

        act(() => void vi.advanceTimersByTime(UNDER_ATTACK_MS))

        expect(onDone).toHaveBeenCalledTimes(1)
    })
})
