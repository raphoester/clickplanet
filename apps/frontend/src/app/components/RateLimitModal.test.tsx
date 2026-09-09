// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import RateLimitModal from "./RateLimitModal.tsx"

afterEach(cleanup)

describe("RateLimitModal", () => {
    it("says what happened, over a backdrop that covers the globe", () => {
        const {container} = render(<RateLimitModal onClose={vi.fn()}/>)

        expect(screen.getByText("You're clicking too fast 🔥")).not.toBeNull()
        expect(container.querySelector(".modal")).not.toBeNull()
    })

    it("closes on the footer button", async () => {
        const onClose = vi.fn()
        const user = userEvent.setup()
        render(<RateLimitModal onClose={onClose}/>)

        await user.click(screen.getByRole("button", {name: "Got it"}))
        expect(onClose).toHaveBeenCalledTimes(1)
    })

    it("stays put when the backdrop is clicked, unlike every other dialog", async () => {
        const onClose = vi.fn()
        const user = userEvent.setup()
        const {container} = render(<RateLimitModal onClose={onClose}/>)

        await user.click(container.querySelector(".modal")!)
        expect(onClose).not.toHaveBeenCalled()
    })

    it("closes on the × and on Escape, like every other dialog", async () => {
        const onClose = vi.fn()
        const user = userEvent.setup()
        render(<RateLimitModal onClose={onClose}/>)

        await user.click(screen.getByRole("button", {name: "Close"}))
        await user.keyboard("{Escape}")
        expect(onClose).toHaveBeenCalledTimes(2)
    })
})
