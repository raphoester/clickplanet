// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import VPNBlockedModal from "./VPNBlockedModal.tsx"

afterEach(cleanup)

describe("VPNBlockedModal", () => {
    it("says what happened, over a backdrop that covers the globe", () => {
        const {container} = render(<VPNBlockedModal onClose={vi.fn()}/>)

        expect(screen.getByText("You can't paint through a VPN 🛡️")).not.toBeNull()
        expect(container.querySelector(".modal")).not.toBeNull()
    })

    /**
     * The one thing the player can act on. A spent bucket refills on its own,
     * this refusal does not, so the copy has to be an instruction.
     */
    it("tells the player to turn the VPN off", () => {
        render(<VPNBlockedModal onClose={vi.fn()}/>)

        expect(screen.getByText(/Turn yours off/)).not.toBeNull()
    })

    it("closes on the footer button", async () => {
        const onClose = vi.fn()
        const user = userEvent.setup()
        render(<VPNBlockedModal onClose={onClose}/>)

        await user.click(screen.getByRole("button", {name: "Got it"}))
        expect(onClose).toHaveBeenCalledTimes(1)
    })

    it("closes on the × and on Escape, like every other dialog", async () => {
        const onClose = vi.fn()
        const user = userEvent.setup()
        render(<VPNBlockedModal onClose={onClose}/>)

        await user.click(screen.getByRole("button", {name: "Close"}))
        await user.keyboard("{Escape}")
        expect(onClose).toHaveBeenCalledTimes(2)
    })
})
