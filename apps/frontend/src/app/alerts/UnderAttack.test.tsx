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
        render(<UnderAttack flag="fr" connect={connect} goThere={vi.fn()} onDone={vi.fn()}/>)

        expect(screen.getByRole("status").textContent).toContain("France is under attack")
        const ping = screen.getByRole("button", {name: "Join Discord and get pinged when France is attacked"})
        expect(ping.textContent).toBe("Get pinged")
        fireEvent.click(ping)

        expect(connect).toHaveBeenCalledTimes(1)
    })

    it("flies to the attack when pressed", () => {
        const goThere = vi.fn()
        render(<UnderAttack flag="fr" connect={vi.fn(async () => undefined)} goThere={goThere} onDone={vi.fn()}/>)

        fireEvent.click(screen.getByRole("button", {name: "Go there"}))

        expect(goThere).toHaveBeenCalledTimes(1)
    })

    it("offers no ping without the Discord sign-in", () => {
        render(<UnderAttack flag="fr" goThere={vi.fn()} onDone={vi.fn()}/>)

        expect(screen.getAllByRole("button").map((button) => button.textContent)).toEqual(["Go there"])
    })

    it("goes away on its own", () => {
        vi.useFakeTimers()
        const onDone = vi.fn()
        render(<UnderAttack flag="fr" connect={vi.fn(async () => undefined)} goThere={vi.fn()} onDone={onDone}/>)

        act(() => void vi.advanceTimersByTime(UNDER_ATTACK_MS))

        expect(onDone).toHaveBeenCalledTimes(1)
    })
})
