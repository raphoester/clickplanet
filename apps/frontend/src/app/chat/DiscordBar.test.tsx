// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {DISCORD_INVITE} from "../../links.ts"
import DiscordBar from "./DiscordBar.tsx"
import {DISCORD_BAR_STORAGE_KEY} from "./useDiscordBar.ts"

const link = () => screen.queryByRole("link", {name: "Join the Discord"})
const hide = () => userEvent.setup().click(screen.getByRole("button", {name: "Hide the Discord link"}))

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

    it("goes when its cross is pressed, and stays gone on the next visit", async () => {
        const {unmount} = render(<DiscordBar/>)

        await hide()
        expect(link()).toBeNull()
        expect(window.localStorage.getItem(DISCORD_BAR_STORAGE_KEY)).toBe("hidden")

        unmount()
        render(<DiscordBar/>)
        expect(link()).toBeNull()
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
