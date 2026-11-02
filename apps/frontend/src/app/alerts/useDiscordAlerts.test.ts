// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {renderHook, waitFor} from "@testing-library/react"
import {AlertsBackend, DiscordLink} from "../../backends/alerts.ts"
import {DiscordAlerts} from "./discordAlerts.ts"
import {useDiscordAlerts} from "./useDiscordAlerts.ts"

const SERVER = "https://discord.com/channels/1/2"

function alertsReading(read: () => Promise<DiscordLink>): {alerts: DiscordAlerts, backend: AlertsBackend} {
    const backend: AlertsBackend = {discordLink: vi.fn(read)}
    return {alerts: new DiscordAlerts(backend, vi.fn(async () => undefined)), backend}
}

afterEach(() => vi.restoreAllMocks())

describe("useDiscordAlerts", () => {
    it("offers to connect while the server is offered and this account is not in it", async () => {
        const {alerts} = alertsReading(async () => ({offered: true, linked: false, serverUrl: SERVER}))

        const {result} = renderHook(() => useDiscordAlerts(alerts, undefined, true))

        await waitFor(() => expect(result.current).toEqual({connect: alerts.connect}))
    })

    it("offers the server's page once this account is in it", async () => {
        const {alerts} = alertsReading(async () => ({offered: true, linked: true, serverUrl: SERVER}))

        const {result} = renderHook(() => useDiscordAlerts(alerts, undefined, true))

        await waitFor(() => expect(result.current).toEqual({serverUrl: SERVER}))
    })

    it("offers nothing when the server is not offered, or Discord's sign-in is not", async () => {
        const off = alertsReading(async () => ({offered: false, linked: false, serverUrl: ""}))
        const noSignIn = alertsReading(async () => ({offered: true, linked: false, serverUrl: SERVER}))

        const first = renderHook(() => useDiscordAlerts(off.alerts, undefined, true))
        const second = renderHook(() => useDiscordAlerts(noSignIn.alerts, undefined, false))

        await waitFor(() => expect(noSignIn.backend.discordLink).toHaveBeenCalled())
        await Promise.resolve()
        expect(first.result.current).toEqual({})
        expect(second.result.current).toEqual({})
    })

    it("offers nothing when the read fails", async () => {
        vi.spyOn(console, "error").mockImplementation(() => {
        })
        const {alerts} = alertsReading(async () => {
            throw new Error("down")
        })

        const {result} = renderHook(() => useDiscordAlerts(alerts, undefined, true))

        await waitFor(() => expect(console.error).toHaveBeenCalled())
        expect(result.current).toEqual({})
    })

    it("reads again when the account changes, and shows the page once the new account is in", async () => {
        const reads: DiscordLink[] = [{offered: true, linked: false, serverUrl: SERVER}, {offered: true, linked: true, serverUrl: SERVER}]
        const {alerts, backend} = alertsReading(async () => reads.shift()!)

        const {result, rerender} = renderHook(({account}) => useDiscordAlerts(alerts, account, true), {initialProps: {account: "guest-1"}})
        await waitFor(() => expect(result.current.connect).toBe(alerts.connect))

        rerender({account: "account-2"})

        await waitFor(() => expect(result.current).toEqual({serverUrl: SERVER}))
        expect(backend.discordLink).toHaveBeenCalledTimes(2)
    })
})
