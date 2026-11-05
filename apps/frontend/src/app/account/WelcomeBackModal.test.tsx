// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import WelcomeBackModal from "./WelcomeBackModal.tsx"

describe("WelcomeBackModal", () => {
    afterEach(cleanup)

    it("says the account is back, and closes on Play", async () => {
        const onClose = vi.fn()
        render(<WelcomeBackModal onClose={onClose}/>)

        const dialog = screen.getByRole("dialog", {name: "Welcome back!"})
        expect(dialog.textContent).toContain("no longer going to be deleted")

        await userEvent.setup().click(screen.getByRole("button", {name: "Play"}))

        expect(onClose).toHaveBeenCalledTimes(1)
    })
})
