// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {DISCORD_INVITE, INSTAGRAM_PROFILE, TIKTOK_PROFILE} from "../../links.ts"
import DiscordBar from "./DiscordBar.tsx"
import {DISCORD_BAR_RETURNS_AFTER_MS, DISCORD_BAR_STORAGE_KEY} from "./discordBar.ts"

const link = () => screen.queryByRole("link", {name: "Join the Discord"})
const hide = () => userEvent.setup().click(screen.getByRole("button", {name: "Hide the links"}))

beforeEach(() => window.localStorage.clear())
afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
})

describe("DiscordBar", () => {
    it("opens the Discord invite in a new tab", () => {
        render(<DiscordBar/>)

        expect(link()?.getAttribute("href")).toBe(DISCORD_INVITE)
        expect(link()?.getAttribute("target")).toBe("_blank")
        expect(link()?.getAttribute("rel")).toBe("noopener noreferrer")
    })

    it.each([
        ["TikTok", TIKTOK_PROFILE],
        ["Instagram", INSTAGRAM_PROFILE],
    ])("links %s beside it, in a new tab", (name, href) => {
        render(<DiscordBar/>)

        const other = screen.getByRole("link", {name})
        expect(other.getAttribute("href")).toBe(href)
        expect(other.getAttribute("target")).toBe("_blank")
        expect(other.getAttribute("rel")).toBe("noopener noreferrer")
    })

    it("goes when its cross is pressed, with the links beside it, and stays gone on a visit soon after", async () => {
        const before = Date.now()
        const {unmount} = render(<DiscordBar/>)

        await hide()
        expect(link()).toBeNull()
        expect(screen.queryByRole("link")).toBeNull()

        const hiddenAt = Number(window.localStorage.getItem(DISCORD_BAR_STORAGE_KEY))
        expect(hiddenAt).toBeGreaterThanOrEqual(before)
        expect(hiddenAt).toBeLessThanOrEqual(Date.now())

        unmount()
        render(<DiscordBar/>)
        expect(link()).toBeNull()
    })

    it("is back on a visit the next day", () => {
        window.localStorage.setItem(DISCORD_BAR_STORAGE_KEY, String(Date.now() - DISCORD_BAR_RETURNS_AFTER_MS))

        render(<DiscordBar/>)

        expect(link()).not.toBeNull()
    })

    it("still goes in a browser that keeps nothing", async () => {
        vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
            throw new Error("storage is off")
        })
        vi.spyOn(console, "error").mockImplementation(() => {
        })
        render(<DiscordBar/>)

        await hide()

        expect(link()).toBeNull()
    })
})
