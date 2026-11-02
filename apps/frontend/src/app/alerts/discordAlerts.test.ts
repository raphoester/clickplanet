import {describe, expect, it, vi} from "vitest"
import {AlertsBackend, AlertsError} from "../../backends/alerts.ts"
import {DiscordAlerts} from "./discordAlerts.ts"

function backendLeavingFor(url: string): AlertsBackend {
    return {
        discordLink: vi.fn(async () => ({offered: true, linked: false})),
        startDiscordLink: vi.fn(async () => url),
        completeDiscordLink: vi.fn(async () => "https://discord.com/channels/1"),
    }
}

describe("DiscordAlerts", () => {
    it("remembers the trip, then sends the browser to Discord's page", async () => {
        const steps: string[] = []
        const alerts = new DiscordAlerts(backendLeavingFor("https://discord.com/oauth2/authorize?x"), {
            remember: () => steps.push("remember"),
            navigate: (url) => steps.push(`go ${url}`),
        })

        await alerts.leave()

        expect(steps).toEqual(["remember", "go https://discord.com/oauth2/authorize?x"])
    })

    it("remembers nothing and stays when the start is refused", async () => {
        const backend = backendLeavingFor("unused")
        backend.startDiscordLink = vi.fn(async () => {
            throw new AlertsError("tooManyTries")
        })
        const remember = vi.fn()
        const navigate = vi.fn()

        await expect(new DiscordAlerts(backend, {remember, navigate}).leave()).rejects.toMatchObject({failure: "tooManyTries"})

        expect(remember).not.toHaveBeenCalled()
        expect(navigate).not.toHaveBeenCalled()
    })
})
