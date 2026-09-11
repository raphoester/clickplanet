// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import CameraButton from "./CameraButton.tsx"

afterEach(cleanup)

const button = () => screen.getByRole("button")

describe("CameraButton", () => {
    it("offers to take one", () => {
        render(<CameraButton onClick={vi.fn()}/>)
        expect(button().textContent).toBe("Photo")
    })

    it("takes one when pressed", async () => {
        const onClick = vi.fn()
        render(<CameraButton onClick={onClick}/>)

        await userEvent.setup().click(button())

        expect(onClick).toHaveBeenCalledTimes(1)
    })

    // One globe, one frame it is captured from: a second press while the first
    // is drawing would capture another and leak the first one's object URL.
    it("refuses a second press while a picture is being drawn", () => {
        render(<CameraButton busy onClick={vi.fn()}/>)

        expect((button() as HTMLButtonElement).disabled).toBe(true)
        expect(button().getAttribute("aria-busy")).toBe("true")
        expect(button().textContent).toBe("One sec…")
    })
})
