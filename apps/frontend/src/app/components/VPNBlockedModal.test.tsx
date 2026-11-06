// @vitest-environment jsdom
import {afterEach, describe, expect, it} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import VPNBlockedModal from "./VPNBlockedModal.tsx"
import {useRefusedClicks} from "../clicks/refusedClicksStore.ts"

afterEach(() => {
    cleanup()
    useRefusedClicks.setState(useRefusedClicks.getInitialState(), true)
})

function blocked() {
    render(<VPNBlockedModal/>)
    act(() => useRefusedClicks.getState().blockVPN())
}

describe("VPNBlockedModal", () => {
    it("says what happened, over a backdrop that covers the globe", () => {
        blocked()

        expect(screen.getByText("You can't paint through a VPN 🛡️")).not.toBeNull()
        expect(document.querySelector(".modal")).not.toBeNull()
    })

    it("tells the player to turn the VPN off", () => {
        blocked()

        expect(screen.getByText(/Turn yours off/)).not.toBeNull()
    })

    it("closes on the footer button, and opens again at the next blocked click", async () => {
        const user = userEvent.setup()
        blocked()

        await user.click(screen.getByRole("button", {name: "Got it"}))
        expect(screen.queryByText(/Turn yours off/)).toBeNull()

        act(() => useRefusedClicks.getState().blockVPN())
        expect(screen.getByText(/Turn yours off/)).not.toBeNull()
    })

    it("closes on the × and on Escape, like every other dialog", async () => {
        const user = userEvent.setup()
        blocked()

        await user.click(screen.getByRole("button", {name: "Close"}))
        expect(useRefusedClicks.getState().vpnBlocked).toBe(false)

        act(() => useRefusedClicks.getState().blockVPN())
        await user.keyboard("{Escape}")
        expect(useRefusedClicks.getState().vpnBlocked).toBe(false)
    })
})
