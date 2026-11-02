// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {AuthError} from "../../backends/account.ts"
import {DISCORD_INVITE} from "../../links.ts"
import DiscordBar from "./DiscordBar.tsx"
import {DISCORD_BAR_RETURNS_AFTER_MS, DISCORD_BAR_STORAGE_KEY} from "./discordBar.ts"

const link = () => screen.queryByRole("link", {name: "Join Discord"})
const hide = () => userEvent.setup().click(screen.getByRole("button", {name: "Hide Discord"}))

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

    it("is the one link it holds: no other network beside it", () => {
        render(<DiscordBar/>)

        expect(screen.getAllByRole("link")).toHaveLength(1)
    })

    it("goes when its cross is pressed, and stays gone on a visit soon after", async () => {
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

    it("joins by Discord's sign-in in place of the invite, and says what for", async () => {
        const connect = vi.fn(async () => undefined)
        render(<DiscordBar discord={{connect}} flagName="France"/>)

        expect(link()).toBeNull()
        const join = screen.getByRole("button", {name: "Join Discord and get an alert when France is under attack"})
        expect(join.textContent).toBe("Join Discord")
        expect(screen.getByText("Get a Discord alert when France is under attack")).toBeDefined()

        await userEvent.setup().click(join)

        expect(connect).toHaveBeenCalledTimes(1)
        expect(join.hasAttribute("disabled")).toBe(true)
    })

    it("says to wait when the sign-in was refused for too many tries, and takes another press", async () => {
        const connect = vi.fn(async () => {
            throw new AuthError("tooManyTries")
        })
        render(<DiscordBar discord={{connect}} flagName="France"/>)

        await userEvent.setup().click(screen.getByRole("button", {name: "Join Discord and get an alert when France is under attack"}))

        const again = await screen.findByRole("button", {name: "Wait a minute"})
        expect(again.hasAttribute("disabled")).toBe(false)
        await userEvent.setup().click(again)
        expect(connect).toHaveBeenCalledTimes(2)
    })

    it("opens the server's page for a player already in it", () => {
        render(<DiscordBar discord={{serverUrl: "https://discord.com/channels/1/2"}} flagName="France"/>)

        expect(link()).toBeNull()
        expect(screen.getByRole("link", {name: "Open Discord"}).getAttribute("href")).toBe("https://discord.com/channels/1/2")
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
