import {describe, expect, it, vi} from "vitest"
import {AlertsBackend} from "../../backends/alerts.ts"
import {DiscordAlerts} from "./discordAlerts.ts"

const backend: AlertsBackend = {
    discordLink: vi.fn(async () => ({offered: true, linked: false, serverUrl: "https://discord.com/channels/1/2"})),
}

describe("DiscordAlerts", () => {
    it("reads the link from the alerts", async () => {
        const alerts = new DiscordAlerts(backend, vi.fn(async () => undefined), false)

        expect(await alerts.read()).toEqual({offered: true, linked: false, serverUrl: "https://discord.com/channels/1/2"})
    })

    it("connects by Discord's sign-in, and says why it could not", async () => {
        const signIn = vi.fn(async () => {
            throw new Error("refused")
        })

        await expect(new DiscordAlerts(backend, signIn, false).connect()).rejects.toThrow("refused")

        expect(signIn).toHaveBeenCalledTimes(1)
    })

    it("starts the sign-in once for a page opened to join", () => {
        const signIn = vi.fn(async () => undefined)
        const alerts = new DiscordAlerts(backend, signIn, true)

        alerts.joinIfAsked()
        alerts.joinIfAsked()

        expect(signIn).toHaveBeenCalledTimes(1)
    })

    it("starts nothing for a page opened to play", () => {
        const signIn = vi.fn(async () => undefined)

        new DiscordAlerts(backend, signIn, false).joinIfAsked()

        expect(signIn).not.toHaveBeenCalled()
    })

    it("tells a sign-in that could not start, and throws nothing", async () => {
        const error = vi.spyOn(console, "error").mockImplementation(() => {
        })
        const alerts = new DiscordAlerts(backend, vi.fn(async () => {
            throw new Error("refused")
        }), true)

        alerts.joinIfAsked()

        await vi.waitFor(() => expect(error).toHaveBeenCalled())
        error.mockRestore()
    })
})
