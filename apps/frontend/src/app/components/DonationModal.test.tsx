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