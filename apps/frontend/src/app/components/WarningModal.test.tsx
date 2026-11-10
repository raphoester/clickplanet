// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, fireEvent, render, screen} from "@testing-library/react"
import WarningModal, {ACKNOWLEDGE_HOLD_MS} from "./WarningModal.tsx"
import {GAME_RULES} from "./BannedModal.tsx"
import {useWarning} from "../warning/warningStore.ts"

beforeEach(() => {
    vi.useFakeTimers()
})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
    vi.restoreAllMocks()
    useWarning.setState(useWarning.getInitialState(), true)
})

function warned() {
    act(() => useWarning.getState().warn())
}

function acknowledging() {
    return {acknowledgeWarning: vi.fn().mockResolvedValue(undefined)}
}

async function hold() {
    fireEvent.pointerDown(screen.getByRole("button", {name: "Hold to confirm"}))
    await act(() => vi.advanceTimersByTimeAsync(ACKNOWLEDGE_HOLD_MS))
}

describe("WarningModal", () => {
    it("asks for one account per player", () => {
        render(<WarningModal warnings={acknowledging()}/>)
        warned()

        expect(screen.getByRole("heading", {name: "One account per player"})).not.toBeNull()
        expect(screen.getByText("You played several accounts to get more clicks. Keep to one account, or your accounts will be paused.")).not.toBeNull()
    })

    it("links to the play fair rules in a new tab", () => {
        render(<WarningModal warnings={acknowledging()}/>)
        warned()

        const link = screen.getByRole("link", {name: "Read the game rules"})
        expect(link.getAttribute("href")).toBe(GAME_RULES)
        expect(link.getAttribute("target")).toBe("_blank")
    })

    it("shows nothing while the account is not warned", () => {
        render(<WarningModal warnings={acknowledging()}/>)

        expect(screen.queryByText("One account per player")).toBeNull()
    })

    it("acknowledges the warning once the button is held, then closes", async () => {
        const warnings = acknowledging()
        render(<WarningModal warnings={warnings}/>)
        warned()

        await hold()

        expect(warnings.acknowledgeWarning).toHaveBeenCalledTimes(1)
        expect(screen.queryByText("One account per player")).toBeNull()
        expect(useWarning.getState().warned).toBe(false)
    })

    it("takes no click: the button must be held", async () => {
        const warnings = acknowledging()
        render(<WarningModal warnings={warnings}/>)
        warned()

        const button = screen.getByRole("button", {name: "Hold to confirm"})
        fireEvent.pointerDown(button)
        fireEvent.pointerUp(button)
        fireEvent.click(button)
        await act(() => vi.advanceTimersByTimeAsync(ACKNOWLEDGE_HOLD_MS))

        expect(warnings.acknowledgeWarning).not.toHaveBeenCalled()
        expect(useWarning.getState().warned).toBe(true)
    })

    it("cannot be closed any other way", () => {
        render(<WarningModal warnings={acknowledging()}/>)
        warned()

        expect(screen.queryByRole("button", {name: "Close"})).toBeNull()
        fireEvent.keyDown(document, {key: "Escape"})
        fireEvent.click(document.querySelector(".modal") as Element)

        expect(useWarning.getState().warned).toBe(true)
        expect(screen.getByText("One account per player")).not.toBeNull()
    })

    it("stays open when the server could not take it, so the player can hold again", async () => {
        const logged = vi.spyOn(console, "error").mockImplementation(() => {})
        const warnings = {acknowledgeWarning: vi.fn().mockRejectedValueOnce(new Error("down")).mockResolvedValueOnce(undefined)}
        render(<WarningModal warnings={warnings}/>)
        warned()

        await hold()

        expect(logged).toHaveBeenCalledTimes(1)
        expect(useWarning.getState().warned).toBe(true)
        expect(screen.getByText("One account per player")).not.toBeNull()

        fireEvent.pointerUp(screen.getByRole("button", {name: "Hold to confirm"}))
        await hold()

        expect(warnings.acknowledgeWarning).toHaveBeenCalledTimes(2)
        expect(useWarning.getState().warned).toBe(false)
    })
})
