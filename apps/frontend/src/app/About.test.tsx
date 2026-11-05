// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import About from "./About.tsx"
import {INSTAGRAM_PROFILE, TIKTOK_PROFILE, YOUTUBE_CHANNEL} from "../links.ts"

afterEach(cleanup)

describe("About", () => {
    it("links to the privacy policy and the terms of service", () => {
        render(<About/>)

        expect(screen.getByRole("link", {name: "Privacy policy"}).getAttribute("href")).toBe("/privacy")
        expect(screen.getByRole("link", {name: "Terms of service"}).getAttribute("href")).toBe("/terms")
    })

    it("joins Discord by its sign-in, and shows no Discord when nothing is offered", () => {
        const {unmount} = render(<About discord={{connect: vi.fn(async () => undefined)}}/>)
        expect(screen.getByRole("button", {name: "Join Discord"})).toBeDefined()

        unmount()
        render(<About discord={{}}/>)
        expect(screen.queryByRole("button", {name: "Join Discord"})).toBeNull()
        expect(screen.queryByRole("link", {name: "Open Discord"})).toBeNull()
    })

    it.each([
        ["TikTok", TIKTOK_PROFILE],
        ["YouTube", YOUTUBE_CHANNEL],
        ["Instagram", INSTAGRAM_PROFILE],
    ])("links the game's %s from links.ts, in a new tab", (name, href) => {
        render(<About/>)

        const link = screen.getByRole("link", {name})
        expect(link.getAttribute("href")).toBe(href)
        expect(link.getAttribute("target")).toBe("_blank")
        expect(link.getAttribute("rel")).toBe("noopener noreferrer")
    })

    it("links back to the home page, past the redirect to the game", () => {
        render(<About/>)

        expect(screen.getByRole("link", {name: "Home page"}).getAttribute("href")).toBe("/#home")
    })
})
