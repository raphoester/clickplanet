// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {renderHook, waitFor} from "@testing-library/react"
import {AlertsBackend, DiscordLink} from "../../backends/alerts.ts"
import {DiscordAlerts} from "./discordAlerts.ts"
import {useDiscordAlerts} from "./useDiscordAlerts.ts"

function alertsReading(read: () => Promise<DiscordLink>): {alerts: DiscordAlerts, backend: AlertsBackend} {
    const backend: AlertsBackend = {
        discordLink: vi.fn(read),
        startDiscordLink: vi.fn(async () => "https://discord.com/oauth2/authorize"),
        completeDiscordLink: vi.fn(async () => "https://discord.com/channels/1"),
    }
    return {alerts: new DiscordAlerts(backend, {navigate: vi.fn(), remember: vi.fn()}), backend}
}

afterEach(() => vi.restoreAllMocks())

describe("useDiscordAlerts", () => {
    it("offers the link while it is offered and not made", async () => {
        const {alerts} = alertsReading(async () => ({offered: true, linked: false}))

        const {result} = renderHook(() => useDiscordAlerts(alerts, undefined))

        await waitFor(() => expect(result.current).toBe(alerts.leave))
    })

    it.each<[string, DiscordLink]>([
        ["not offered", {offered: false, linked: false}],
        ["already made", {offered: true, linked: true}],
    ])("offers nothing when the link is %s", async (_, link) => {
        const {alerts, backend} = alertsReading(async () => link)

        const {result} = renderHook(() => useDiscordAlerts(alerts, undefined))

        await waitFor(() => expect(backend.discordLink).toHaveBeenCalled())
        await Promise.resolve()
        expect(result.current).toBeUndefined()
    })

    it("offers nothing when the read fails", async () => {
        vi.spyOn(console, "error").mockImplementation(() => {
        })
        const {alerts} = alertsReading(async () => {
            throw new Error("down")
        })

        const {result} = renderHook(() => useDiscordAlerts(alerts, undefined))

        await waitFor(() => expect(console.error).toHaveBeenCalled())
        expect(result.current).toBeUndefined()
    })

    it("reads again when the account changes, and drops an offer the new account has made", async () => {
        const reads = [{offered: true, linked: false}, {offered: true, linked: true}]
        const {alerts, backend} = alertsReading(async () => reads.shift()!)

        const {result, rerender} = renderHook(({account}) => useDiscordAlerts(alerts, account), {initialProps: {account: "guest-1"}})
        await waitFor(() => expect(result.current).toBe(alerts.leave))

        rerender({account: "account-2"})

        await waitFor(() => expect(result.current).toBeUndefined())
        expect(backend.discordLink).toHaveBeenCalledTimes(2)
    })
})
