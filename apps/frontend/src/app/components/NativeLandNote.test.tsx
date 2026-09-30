// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import NativeLandNote, {NATIVE_LAND_NOTE_MS} from "./NativeLandNote.tsx"

describe("NativeLandNote", () => {
    beforeEach(() => vi.useFakeTimers())
    afterEach(() => {
        cleanup()
        vi.useRealTimers()
    })

    it("says whose native land it was and that one more click takes it", () => {
        render(<NativeLandNote ground="pl" onDone={() => {}}/>)

        expect(screen.getByRole("status").textContent).toBe("Poland's native land takes two clicks. One more to take it.")
    })

    it("takes itself away, however often its parent re-renders", () => {
        const onDone = vi.fn()
        const {rerender} = render(<NativeLandNote ground="pl" onDone={onDone}/>)

        for (let tick = 0; tick * 500 < NATIVE_LAND_NOTE_MS; tick++) {
            act(() => void vi.advanceTimersByTime(500))
            rerender(<NativeLandNote ground="pl" onDone={() => onDone()}/>)
        }

        expect(onDone).toHaveBeenCalledTimes(1)
    })
})
