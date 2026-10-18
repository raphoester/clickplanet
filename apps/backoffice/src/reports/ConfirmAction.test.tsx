// @vitest-environment jsdom
import {cleanup, fireEvent, render, screen} from "@testing-library/react"
import {afterEach, describe, expect, it, vi} from "vitest"
import {ConfirmAction} from "./ConfirmAction.tsx"

afterEach(cleanup)

function renderAction(run: () => Promise<string>) {
    render(
        <ConfirmAction
            label="Mute"
            question="Mute Ana for 1 hour?"
            run={run}
            said={value => `Said: ${value}`}
            refusal={() => "Refused in plain words."}
        />,
    )
}

describe("ConfirmAction", () => {
    it("asks before it runs anything", () => {
        const run = vi.fn(() => Promise.resolve("ok"))
        renderAction(run)

        fireEvent.click(screen.getByRole("button", {name: "Mute"}))

        expect(screen.getByText("Mute Ana for 1 hour?")).toBeTruthy()
        expect(run).not.toHaveBeenCalled()
    })

    it("runs nothing when cancelled", () => {
        const run = vi.fn(() => Promise.resolve("ok"))
        renderAction(run)

        fireEvent.click(screen.getByRole("button", {name: "Mute"}))
        fireEvent.click(screen.getByRole("button", {name: "Cancel"}))

        expect(run).not.toHaveBeenCalled()
        expect(screen.queryByText("Mute Ana for 1 hour?")).toBeNull()
    })

    it("runs once confirmed, and says what the server answered", async () => {
        const run = vi.fn(() => Promise.resolve("muted until noon"))
        renderAction(run)

        fireEvent.click(screen.getByRole("button", {name: "Mute"}))
        fireEvent.click(screen.getByRole("button", {name: "Yes, mute"}))

        expect(await screen.findByText("Said: muted until noon")).toBeTruthy()
        expect(run).toHaveBeenCalledTimes(1)
    })

    it("shows the refusal in plain words", async () => {
        renderAction(() => Promise.reject(new Error("boom")))

        fireEvent.click(screen.getByRole("button", {name: "Mute"}))
        fireEvent.click(screen.getByRole("button", {name: "Yes, mute"}))

        expect(await screen.findByRole("alert")).toHaveProperty("textContent", "Refused in plain words.")
    })
})
