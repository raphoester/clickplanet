// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {AuthError} from "../../backends/account.ts"
import DiscordBar from "./DiscordBar.tsx"
import {DISCORD_BAR_RETURNS_AFTER_MS, DISCORD_BAR_STORAGE_KEY} from "./discordBar.ts"

const SERVER = "https://discord.com/channels/1/2"
const IN_SERVER = {serverUrl: SERVER}
const link = () => screen.queryByRole("link", {name: "Open Discord"})
const hide = () => userEvent.setup().click(screen.getByRole("button", {name: "Hide Discord"}))

beforeEach(() => window.localStorage.clear())
afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
})

describe("DiscordBar", () => {
    it("shows nothing while the game offers no way in: an invite would join with no flag's role", () => {
        const {container} = render(<DiscordBar/>)
        expect(container.innerHTML).toBe("")

        cleanup()
        const offered = render(<DiscordBar discord={{}}/>)
        expect(offered.container.innerHTML).toBe("")
    })

    it("is the one link it holds: no other network beside it", () => {
        render(<DiscordBar discord={IN_SERVER}/>)

        expect(screen.getAllByRole("link")).toHaveLength(1)
    })

    it("goes when its cross is pressed, and stays gone on a visit soon after", async () => {
        const before = Date.now()
        const {unmount} = render(<DiscordBar discord={IN_SERVER}/>)

        await hide()
        expect(link()).toBeNull()
        expect(screen.queryByRole("link")).toBeNull()

        const hiddenAt = Number(window.localStorage.getItem(DISCORD_BAR_STORAGE_KEY))
        expect(hiddenAt).toBeGreaterThanOrEqual(before)
        expect(hiddenAt).toBeLessThanOrEqual(Date.now())

        unmount()
        render(<DiscordBar discord={IN_SERVER}/>)
        expect(link()).toBeNull()
    })

    it("is back on a visit the next day", () => {
        window.localStorage.setItem(DISCORD_BAR_STORAGE_KEY, String(Date.now() - DISCORD_BAR_RETURNS_AFTER_MS))

        render(<DiscordBar discord={IN_SERVER}/>)

        expect(link()).not.toBeNull()
    })

    it("joins by Discord's sign-in, and says what for", async () => {
        const connect = vi.fn(async () => undefined)
        render(<DiscordBar discord={{connect}} flagName="France"/>)

        expect(link()).toBeNull()
        const join = screen.getByRole("button", {name: "Join Discord and get pinged when France is attacked"})
        expect(join.textContent).toBe("Join Discord")
        expect(screen.getByText("Get pinged when France is attacked")).toBeDefined()

        await userEvent.setup().click(join)

        expect(connect).toHaveBeenCalledTimes(1)
        expect(join.hasAttribute("disabled")).toBe(true)
    })

    it("says to wait when the sign-in was refused for too many tries, and takes another press", async () => {
        const connect = vi.fn(async () => {
            throw new AuthError("tooManyTries")
        })
        render(<DiscordBar discord={{connect}} flagName="France"/>)

        await userEvent.setup().click(screen.getByRole("button", {name: "Join Discord and get pinged when France is attacked"}))

        const again = await screen.findByRole("button", {name: "Wait a minute"})
        expect(again.hasAttribute("disabled")).toBe(false)
        await userEvent.setup().click(again)
        expect(connect).toHaveBeenCalledTimes(2)
    })

    it("opens the server's page in a new tab for a player already in it", () => {
        render(<DiscordBar discord={IN_SERVER} flagName="France"/>)

        expect(link()?.getAttribute("href")).toBe(SERVER)
        expect(link()?.getAttribute("target")).toBe("_blank")
        expect(link()?.getAttribute("rel")).toBe("noopener noreferrer")
    })

    it("still goes in a browser that keeps nothing", async () => {
        vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
            throw new Error("storage is off")
        })
        vi.spyOn(console, "error").mockImplementation(() => {
        })
        render(<DiscordBar discord={IN_SERVER}/>)

        await hide()

        expect(link()).toBeNull()
    })
})
