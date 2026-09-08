// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import DonationModal from "./DonationModal.tsx"

const shown = () => screen.queryByText("Do you like ClickPlanet ?")

beforeEach(() => vi.restoreAllMocks())
afterEach(cleanup)

describe("DonationModal", () => {
    it("appears on a high roll and takes the whole screen", () => {
        vi.spyOn(Math, "random").mockReturnValue(0.9)
        const {container} = render(<DonationModal/>)

        expect(shown()).not.toBeNull()
        expect(container.querySelector(".modal")).not.toBeNull()
    })

    it("stays away on a low roll", () => {
        vi.spyOn(Math, "random").mockReturnValue(0.1)
        render(<DonationModal/>)
        expect(shown()).toBeNull()
    })

    it("closes on the close button", async () => {
        vi.spyOn(Math, "random").mockReturnValue(0.9)
        const user = userEvent.setup()
        render(<DonationModal/>)

        await user.click(screen.getByRole("button", {name: "Close"}))
        expect(shown()).toBeNull()
    })

    it("closes on the backdrop, but not on the panel", async () => {
        vi.spyOn(Math, "random").mockReturnValue(0.9)
        const user = userEvent.setup()
        const {container} = render(<DonationModal/>)

        await user.click(screen.getByText("Do you like ClickPlanet ?"))
        expect(shown()).not.toBeNull()

        await user.click(container.querySelector(".modal")!)
        expect(shown()).toBeNull()
    })

    it("announces itself as a modal dialog, labelled by its title", () => {
        vi.spyOn(Math, "random").mockReturnValue(0.9)
        render(<DonationModal/>)

        const dialog = screen.getByRole("dialog", {name: "Dear earthlings"})
        expect(dialog.getAttribute("aria-modal")).toBe("true")
    })

    it("closes on Escape", async () => {
        vi.spyOn(Math, "random").mockReturnValue(0.9)
        const user = userEvent.setup()
        render(<DonationModal/>)

        await user.keyboard("{Escape}")
        expect(shown()).toBeNull()
    })

    it("moves focus into the dialog when it opens", () => {
        vi.spyOn(Math, "random").mockReturnValue(0.9)
        const {container} = render(<DonationModal/>)

        expect(container.querySelector(".modal-content")!.contains(document.activeElement)).toBe(true)
    })

    /** The globe and the menu are still in the tab order behind the backdrop. */
    it("keeps Tab inside the dialog", async () => {
        vi.spyOn(Math, "random").mockReturnValue(0.9)
        const user = userEvent.setup()
        const {container} = render(<DonationModal/>)

        const panel = container.querySelector(".modal-content")!
        for (let i = 0; i < 6; i++) {
            await user.tab()
            expect(panel.contains(document.activeElement)).toBe(true)
        }

        await user.tab({shift: true})
        expect(panel.contains(document.activeElement)).toBe(true)
    })

    it("hands focus back to whatever had it when the dialog closes", async () => {
        vi.spyOn(Math, "random").mockReturnValue(0.9)
        const user = userEvent.setup()

        const opener = document.createElement("button")
        opener.textContent = "opener"
        document.body.appendChild(opener)
        opener.focus()

        render(<DonationModal/>)
        await user.click(screen.getByRole("button", {name: "Close"}))

        expect(document.activeElement).toBe(opener)
        opener.remove()
    })

    /**
     * The roll used to be a bare `Math.random()` in App's JSX. React may render
     * a component more than once for one commit, so it could flip on re-render.
     */
    it("rolls once, and does not re-roll on re-render", () => {
        const random = vi.spyOn(Math, "random").mockReturnValue(0.9)
        const {rerender} = render(<DonationModal/>)

        random.mockReturnValue(0.1)
        rerender(<DonationModal/>)
        rerender(<DonationModal/>)

        expect(shown()).not.toBeNull()
        expect(random).toHaveBeenCalledTimes(1)
    })
})