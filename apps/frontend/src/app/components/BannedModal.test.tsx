// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import BannedModal, {GAME_RULES} from "./BannedModal.tsx"
import {describeEnd} from "../../domain/endTime.ts"
import terms from "../../../terms.html?raw"

afterEach(cleanup)

describe("BannedModal", () => {
    it("says until when the clicks are paused", () => {
        const until = Date.now() + 3600_000
        render(<BannedModal until={until} onClose={vi.fn()}/>)

        expect(screen.getByText(`Clicks from your account or network are paused until ${describeEnd(until, Date.now())}.`)).not.toBeNull()
    })

    it("links to the play fair rules in a new tab", () => {
        render(<BannedModal until={Date.now() + 3600_000} onClose={vi.fn()}/>)

        const link = screen.getByRole("link", {name: "Read the game rules"})
        expect(link.getAttribute("href")).toBe(GAME_RULES)
        expect(link.getAttribute("target")).toBe("_blank")
    })

    it("points at a section the terms page has", () => {
        const [page, anchor] = GAME_RULES.split("#")

        expect(page).toBe("/terms")
        expect(terms).toContain(`id="${anchor}"`)
    })

    it("closes on the footer button, the × and Escape", async () => {
        const onClose = vi.fn()
        const user = userEvent.setup()
        render(<BannedModal until={Date.now() + 3600_000} onClose={onClose}/>)

        await user.click(screen.getByRole("button", {name: "Got it"}))
        await user.click(screen.getByRole("button", {name: "Close"}))
        await user.keyboard("{Escape}")
        expect(onClose).toHaveBeenCalledTimes(3)
    })
})
