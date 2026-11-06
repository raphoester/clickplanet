// @vitest-environment jsdom
import {afterEach, describe, expect, it} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import BannedModal, {GAME_RULES} from "./BannedModal.tsx"
import {useRefusedClicks} from "../clicks/refusedClicksStore.ts"
import {describeEnd} from "../../domain/endTime.ts"
import terms from "../../../terms.html?raw"

afterEach(() => {
    cleanup()
    useRefusedClicks.setState(useRefusedClicks.getInitialState(), true)
})

function banned(until = Date.now() + 3600_000) {
    act(() => useRefusedClicks.getState().ban(until))
}

describe("BannedModal", () => {
    it("says until when the clicks are paused", () => {
        const until = Date.now() + 3600_000
        render(<BannedModal/>)
        banned(until)

        expect(screen.getByText(`Clicks from your account or network are paused until ${describeEnd(until, Date.now())}.`)).not.toBeNull()
    })

    it("links to the play fair rules in a new tab", () => {
        render(<BannedModal/>)
        banned()

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
        const user = userEvent.setup()
        render(<BannedModal/>)

        for (const close of [() => user.click(screen.getByRole("button", {name: "Got it"})),
            () => user.click(screen.getByRole("button", {name: "Close"})),
            () => user.keyboard("{Escape}")]) {
            banned()
            await close()
            expect(useRefusedClicks.getState().bannedUntil).toBeUndefined()
            expect(screen.queryByText("Clicks paused")).toBeNull()
        }
    })
})
