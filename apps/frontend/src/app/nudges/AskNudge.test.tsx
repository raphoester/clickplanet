// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {TIKTOK_PROFILE, YOUTUBE_CHANNEL} from "../../links.ts"
import AskNudge from "./AskNudge.tsx"

afterEach(() => {
    cleanup()
})

function coffee() {
    const onOpen = vi.fn()
    const onClose = vi.fn()
    render(<AskNudge ask="coffee" onOpen={onOpen} onClose={onClose}/>)
    return {onOpen, onClose}
}

describe("AskNudge", () => {
    it("asks for a coffee in a modal dialog, labelled by its title, over the whole screen", () => {
        coffee()

        const dialog = screen.getByRole("dialog", {name: "Like ClickPlanet?"})
        expect(dialog.getAttribute("aria-modal")).toBe("true")
        expect(document.querySelector(".modal")).not.toBeNull()
        expect(screen.getByRole("link", {name: "Buy me a coffee"})).toBeDefined()
    })

    it("closes on the close button, on Escape and on the backdrop, but not on the panel", async () => {
        const user = userEvent.setup()
        const {onClose} = coffee()

        await user.click(screen.getByText("It's free and open source."))
        expect(onClose).not.toHaveBeenCalled()

        await user.click(screen.getByRole("button", {name: "Close"}))
        await user.keyboard("{Escape}")
        await user.click(document.querySelector(".modal")!)
        expect(onClose).toHaveBeenCalledTimes(3)
    })

    it("keeps the focus inside the dialog, and hands it back once closed", async () => {
        const user = userEvent.setup()
        const opener = document.createElement("button")
        opener.textContent = "opener"
        document.body.appendChild(opener)
        opener.focus()

        const {unmount} = render(<AskNudge ask="coffee" onOpen={vi.fn()} onClose={vi.fn()}/>)
        const panel = document.querySelector(".modal-content")!
        expect(panel.contains(document.activeElement)).toBe(true)
        for (let i = 0; i < 6; i++) {
            await user.tab()
            expect(panel.contains(document.activeElement)).toBe(true)
        }
        await user.tab({shift: true})
        expect(panel.contains(document.activeElement)).toBe(true)

        unmount()
        expect(document.activeElement).toBe(opener)
        opener.remove()
    })

    it("tells that the coffee's link was opened", async () => {
        const {onOpen, onClose} = coffee()

        await userEvent.setup().click(screen.getByRole("link", {name: "Buy me a coffee"}))

        expect(onOpen).toHaveBeenCalledOnce()
        expect(onClose).not.toHaveBeenCalled()
    })

    it("asks for a follow on TikTok and a subscribe on YouTube, each with its link", async () => {
        const onOpen = vi.fn()
        const {rerender} = render(<AskNudge ask="tiktok" onOpen={onOpen} onClose={vi.fn()}/>)
        expect(screen.getByRole("dialog", {name: "ClickPlanet on TikTok"})).toBeDefined()
        const follow = screen.getByRole("link", {name: "Follow @clickplanetlol"})
        expect(follow.getAttribute("href")).toBe(TIKTOK_PROFILE)
        await userEvent.setup().click(follow)
        expect(onOpen).toHaveBeenCalledOnce()

        rerender(<AskNudge ask="youtube" onOpen={onOpen} onClose={vi.fn()}/>)
        expect(screen.getByRole("dialog", {name: "ClickPlanet on YouTube"})).toBeDefined()
        expect(screen.getByRole("link", {name: "Subscribe"}).getAttribute("href")).toBe(YOUTUBE_CHANNEL)
    })
})
