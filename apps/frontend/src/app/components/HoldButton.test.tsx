// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, fireEvent, render, screen} from "@testing-library/react"
import HoldButton from "./HoldButton.tsx"

const HOLD_MS = 2_000

beforeEach(() => {
    vi.useFakeTimers()
})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
})

function rendered(onHeld = vi.fn(), disabled = false) {
    render(<HoldButton label="Hold to confirm" holdMs={HOLD_MS} disabled={disabled} onHeld={onHeld}/>)
    return {onHeld, button: screen.getByRole("button", {name: "Hold to confirm"})}
}

describe("HoldButton", () => {
    it("fires once the button is held the whole time", () => {
        const {onHeld, button} = rendered()

        fireEvent.pointerDown(button)
        act(() => vi.advanceTimersByTime(HOLD_MS - 1))
        expect(onHeld).not.toHaveBeenCalled()

        act(() => vi.advanceTimersByTime(1))
        expect(onHeld).toHaveBeenCalledTimes(1)
    })

    it("does nothing on a click", () => {
        const {onHeld, button} = rendered()

        fireEvent.pointerDown(button)
        fireEvent.pointerUp(button)
        fireEvent.click(button)
        act(() => vi.advanceTimersByTime(10 * HOLD_MS))

        expect(onHeld).not.toHaveBeenCalled()
    })

    it("starts again from zero when released early", () => {
        const {onHeld, button} = rendered()

        fireEvent.pointerDown(button)
        act(() => vi.advanceTimersByTime(HOLD_MS - 100))
        fireEvent.pointerUp(button)
        fireEvent.pointerDown(button)
        act(() => vi.advanceTimersByTime(HOLD_MS - 100))
        expect(onHeld).not.toHaveBeenCalled()

        act(() => vi.advanceTimersByTime(100))
        expect(onHeld).toHaveBeenCalledTimes(1)
    })

    it("stops when the finger slides off the button", () => {
        const {onHeld, button} = rendered()

        fireEvent.pointerDown(button)
        fireEvent.pointerLeave(button)
        act(() => vi.advanceTimersByTime(HOLD_MS))

        expect(onHeld).not.toHaveBeenCalled()
    })

    it("takes the space bar held down, and not its repeats", () => {
        const {onHeld, button} = rendered()

        fireEvent.keyDown(button, {key: " "})
        act(() => vi.advanceTimersByTime(HOLD_MS / 2))
        fireEvent.keyDown(button, {key: " ", repeat: true})
        act(() => vi.advanceTimersByTime(HOLD_MS / 2))

        expect(onHeld).toHaveBeenCalledTimes(1)
    })

    it("stops when the key is let go", () => {
        const {onHeld, button} = rendered()

        fireEvent.keyDown(button, {key: "Enter"})
        fireEvent.keyUp(button, {key: "Enter"})
        act(() => vi.advanceTimersByTime(HOLD_MS))

        expect(onHeld).not.toHaveBeenCalled()
    })

    it("cannot be held while disabled", () => {
        const {onHeld, button} = rendered(vi.fn(), true)

        fireEvent.pointerDown(button)
        act(() => vi.advanceTimersByTime(HOLD_MS))

        expect(onHeld).not.toHaveBeenCalled()
    })
})
