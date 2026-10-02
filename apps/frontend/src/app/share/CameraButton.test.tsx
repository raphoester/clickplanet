// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import CameraButton from "./CameraButton.tsx"

afterEach(cleanup)

const button = () => screen.getByRole("button", {name: "Take a picture of your planet"})

describe("CameraButton", () => {
    it("offers to take one", () => {
        render(<CameraButton onClick={vi.fn()}/>)
        expect(screen.getByText("Take a picture")).toBeTruthy()
    })

    it("takes one when pressed", async () => {
        const onClick = vi.fn()
        render(<CameraButton onClick={onClick}/>)

        await userEvent.setup().click(button())

        expect(onClick).toHaveBeenCalledTimes(1)
    })

    it("refuses a second press while a picture is being drawn", () => {
        render(<CameraButton busy onClick={vi.fn()}/>)

        expect((button() as HTMLButtonElement).disabled).toBe(true)
        expect(button().getAttribute("aria-busy")).toBe("true")
    })

    it("keeps its label while it works, rather than resizing under the cursor", () => {
        const {rerender} = render(<CameraButton onClick={vi.fn()}/>)
        const resting = button().textContent

        rerender(<CameraButton busy onClick={vi.fn()}/>)

        expect(button().textContent).toBe(resting)
    })
})
