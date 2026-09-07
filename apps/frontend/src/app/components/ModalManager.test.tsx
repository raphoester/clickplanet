// @vitest-environment jsdom
import {afterEach, describe, expect, it} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import ModalManager from "./ModalManager.tsx"

const open = () => screen.queryByText("the contents")

afterEach(cleanup)

function setup(props: Partial<React.ComponentProps<typeof ModalManager>> = {}) {
    return render(
        <ModalManager modalTitle="A title" buttonProps={{text: "Open"}} {...props}>
            <p>the contents</p>
        </ModalManager>,
    )
}

describe("ModalManager", () => {
    it("starts closed and shows only its button", () => {
        setup()
        expect(open()).toBeNull()
        expect(screen.getByRole("button", {name: "Open"})).toBeDefined()
    })

    it("can start open", () => {
        setup({openByDefault: true})
        expect(open()).not.toBeNull()
    })

    it("opens on the button, and renders its children and title", async () => {
        const user = userEvent.setup()
        setup()

        await user.click(screen.getByRole("button", {name: "Open"}))

        expect(open()).not.toBeNull()
        expect(screen.getByText("A title")).toBeDefined()
    })

    it("closes on the close button", async () => {
        const user = userEvent.setup()
        setup({openByDefault: true, closeButtonText: "Back"})

        await user.click(screen.getByRole("button", {name: "Back"}))
        expect(open()).toBeNull()
    })

    it("closes when the backdrop is clicked but not the panel", async () => {
        const user = userEvent.setup()
        const {container} = setup({openByDefault: true})

        await user.click(screen.getByText("the contents"))
        expect(open()).not.toBeNull()

        await user.click(container.querySelector(".modal")!)
        expect(open()).toBeNull()
    })
})
