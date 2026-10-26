// @vitest-environment jsdom
import {afterEach, describe, expect, it} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import About from "./About.tsx"
import {DISCORD_INVITE, INSTAGRAM_PROFILE, TIKTOK_PROFILE, YOUTUBE_CHANNEL} from "../links.ts"

afterEach(cleanup)

describe("About", () => {
    it("links to the privacy policy and the terms of service", () => {
        render(<About/>)

        expect(screen.getByRole("link", {name: "Privacy policy"}).getAttribute("href")).toBe("/privacy")
        expect(screen.getByRole("link", {name: "Terms of service"}).getAttribute("href")).toBe("/terms")
    })

    it.each([
        ["Discord", DISCORD_INVITE],
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
