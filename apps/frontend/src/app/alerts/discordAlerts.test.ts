import {describe, expect, it, vi} from "vitest"
import {AlertsBackend} from "../../backends/alerts.ts"
import {DiscordAlerts} from "./discordAlerts.ts"

const backend: AlertsBackend = {
    discordLink: vi.fn(async () => ({offered: true, linked: false, serverUrl: "https://discord.com/channels/1/2"})),
}

describe("DiscordAlerts", () => {
    it("reads the link from the alerts", async () => {
        const alerts = new DiscordAlerts(backend, vi.fn(async () => undefined))

        expect(await alerts.read()).toEqual({offered: true, linked: false, serverUrl: "https://discord.com/channels/1/2"})
    })

    it("connects by Discord's sign-in, and says why it could not", async () => {
        const signIn = vi.fn(async () => {
            throw new Error("refused")
        })

        await expect(new DiscordAlerts(backend, signIn).connect()).rejects.toThrow("refused")

        expect(signIn).toHaveBeenCalledTimes(1)
    })
})
