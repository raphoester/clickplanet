// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen, waitFor} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import WarningModal from "./WarningModal.tsx"
import {GAME_RULES} from "./BannedModal.tsx"
import {useRefusedClicks} from "../clicks/refusedClicksStore.ts"

afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
    useRefusedClicks.setState(useRefusedClicks.getInitialState(), true)
})

function warned() {
    act(() => useRefusedClicks.getState().warn())
}

function acknowledging() {
    return {acknowledgeWarning: vi.fn().mockResolvedValue(undefined)}
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

    it("acknowledges the warning on Got it, then closes", async () => {
        const user = userEvent.setup()
        const warnings = acknowledging()
        render(<WarningModal warnings={warnings}/>)
        warned()

        await user.click(screen.getByRole("button", {name: "Got it"}))

        expect(warnings.acknowledgeWarning).toHaveBeenCalledTimes(1)
        await waitFor(() => expect(screen.queryByText("One account per player")).toBeNull())
        expect(useRefusedClicks.getState().warned).toBe(false)
    })

    it("stays open when the server could not take it, so the player can press again", async () => {
        const logged = vi.spyOn(console, "error").mockImplementation(() => {})
        const user = userEvent.setup()
        const warnings = {acknowledgeWarning: vi.fn().mockRejectedValueOnce(new Error("down")).mockResolvedValueOnce(undefined)}
        render(<WarningModal warnings={warnings}/>)
        warned()

        await user.click(screen.getByRole("button", {name: "Got it"}))
        await waitFor(() => expect(logged).toHaveBeenCalledTimes(1))

        expect(useRefusedClicks.getState().warned).toBe(true)
        expect(screen.getByText("One account per player")).not.toBeNull()

        await user.click(screen.getByRole("button", {name: "Got it"}))

        expect(warnings.acknowledgeWarning).toHaveBeenCalledTimes(2)
        await waitFor(() => expect(useRefusedClicks.getState().warned).toBe(false))
    })

    it("closes on the × and Escape without acknowledging", async () => {
        const user = userEvent.setup()
        const warnings = acknowledging()
        render(<WarningModal warnings={warnings}/>)

        for (const close of [() => user.click(screen.getByRole("button", {name: "Close"})),
            () => user.keyboard("{Escape}")]) {
            warned()
            await close()
            expect(useRefusedClicks.getState().warned).toBe(false)
            expect(screen.queryByText("One account per player")).toBeNull()
        }
        expect(warnings.acknowledgeWarning).not.toHaveBeenCalled()
    })
})
