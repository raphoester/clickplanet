// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, renderHook, waitFor} from "@testing-library/react"
import {AlertsBackend, DiscordLink} from "../../backends/alerts.ts"
import {DiscordAlerts} from "./discordAlerts.ts"
import {useDiscordAlerts} from "./useDiscordAlerts.ts"

const SERVER = "https://discord.com/channels/1/2"

function alertsReading(read: () => Promise<DiscordLink>, joinAsked = false) {
    const backend: AlertsBackend = {discordLink: vi.fn(read)}
    const signIn = vi.fn(async () => undefined)
    return {alerts: new DiscordAlerts(backend, signIn, joinAsked), backend, signIn}
}

afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
})

const UNLINKED: DiscordLink = {offered: true, linked: false, serverUrl: SERVER}
const LINKED: DiscordLink = {offered: true, linked: true, serverUrl: SERVER}

async function waited(ms: number) {
    await act(async () => {
        await vi.advanceTimersByTimeAsync(ms)
    })
}

describe("useDiscordAlerts", () => {
    it("offers to connect while the server is offered and this account is not in it", async () => {
        const {alerts} = alertsReading(async () => ({offered: true, linked: false, serverUrl: SERVER}))

        const {result} = renderHook(() => useDiscordAlerts(alerts, undefined, true, false))

        await waitFor(() => expect(result.current).toEqual({connect: alerts.connect}))
    })

    it("starts the sign-in at the first offer to connect, for a page opened to join", async () => {
        const {alerts, signIn} = alertsReading(async () => UNLINKED, true)

        const {result, rerender} = renderHook(({connectable}) => useDiscordAlerts(alerts, "guest-1", connectable, false),
            {initialProps: {connectable: false}})
        await waitFor(() => expect(result.current).toEqual({}))
        expect(signIn).not.toHaveBeenCalled()

        rerender({connectable: true})

        await waitFor(() => expect(signIn).toHaveBeenCalledTimes(1))
        expect(result.current).toEqual({connect: alerts.connect})
    })

    it("starts no sign-in for a page opened to join by an account in the server already", async () => {
        const {alerts, backend, signIn} = alertsReading(async () => LINKED, true)

        const {result} = renderHook(() => useDiscordAlerts(alerts, "account-1", true, false))

        await waitFor(() => expect(result.current).toEqual({serverUrl: SERVER}))
        expect(backend.discordLink).toHaveBeenCalled()
        expect(signIn).not.toHaveBeenCalled()
    })

    it("offers the server's page once this account is in it", async () => {
        const {alerts} = alertsReading(async () => ({offered: true, linked: true, serverUrl: SERVER}))

        const {result} = renderHook(() => useDiscordAlerts(alerts, undefined, true, false))

        await waitFor(() => expect(result.current).toEqual({serverUrl: SERVER}))
    })

    it("offers nothing when the server is not offered, or Discord's sign-in is not", async () => {
        const off = alertsReading(async () => ({offered: false, linked: false, serverUrl: ""}))
        const noSignIn = alertsReading(async () => ({offered: true, linked: false, serverUrl: SERVER}))

        const first = renderHook(() => useDiscordAlerts(off.alerts, undefined, true, false))
        const second = renderHook(() => useDiscordAlerts(noSignIn.alerts, undefined, false, false))

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

        const {result} = renderHook(() => useDiscordAlerts(alerts, undefined, true, false))

        await waitFor(() => expect(console.error).toHaveBeenCalled())
        expect(result.current).toEqual({})
    })

    it("reads again when the account changes, and shows the page once the new account is in", async () => {
        const reads: DiscordLink[] = [{offered: true, linked: false, serverUrl: SERVER}, {offered: true, linked: true, serverUrl: SERVER}]
        const {alerts, backend} = alertsReading(async () => reads.shift()!)

        const {result, rerender} = renderHook(({account}) => useDiscordAlerts(alerts, account, true, false), {initialProps: {account: "guest-1"}})
        await waitFor(() => expect(result.current.connect).toBe(alerts.connect))

        rerender({account: "account-2"})

        await waitFor(() => expect(result.current).toEqual({serverUrl: SERVER}))
        expect(backend.discordLink).toHaveBeenCalledTimes(2)
    })

    describe("after a Discord sign-in, which joins the server a moment later", () => {
        it("reads again after 1 s, then 2 s more, until the account is in the server", async () => {
            vi.useFakeTimers()
            const reads = [UNLINKED, UNLINKED, LINKED]
            const {alerts, backend} = alertsReading(async () => reads.shift() ?? LINKED)

            const {result} = renderHook(() => useDiscordAlerts(alerts, "account-1", true, true))
            await waited(0)
            expect(result.current).toEqual({connect: alerts.connect})

            await waited(999)
            expect(backend.discordLink).toHaveBeenCalledTimes(1)
            await waited(1)
            expect(backend.discordLink).toHaveBeenCalledTimes(2)
            expect(result.current).toEqual({connect: alerts.connect})

            await waited(2_000)
            expect(backend.discordLink).toHaveBeenCalledTimes(3)
            expect(result.current).toEqual({serverUrl: SERVER})

            await waited(60_000)
            expect(backend.discordLink).toHaveBeenCalledTimes(3)
        })

        it("gives up after the read 4 s after the one before", async () => {
            vi.useFakeTimers()
            const {alerts, backend} = alertsReading(async () => UNLINKED)

            const {result} = renderHook(() => useDiscordAlerts(alerts, "account-1", true, true))
            await waited(1_000 + 2_000 + 4_000)
            expect(backend.discordLink).toHaveBeenCalledTimes(4)

            await waited(60_000)
            expect(backend.discordLink).toHaveBeenCalledTimes(4)
            expect(result.current, "the bar offers to join again").toEqual({connect: alerts.connect})
        })

        it("reads once when no Discord sign-in came back, and never again once the page is gone", async () => {
            vi.useFakeTimers()
            const plain = alertsReading(async () => UNLINKED)
            const gone = alertsReading(async () => UNLINKED)

            renderHook(() => useDiscordAlerts(plain.alerts, "account-1", true, false))
            const {unmount} = renderHook(() => useDiscordAlerts(gone.alerts, "account-1", true, true))
            await waited(0)
            unmount()
            await waited(60_000)

            expect(plain.backend.discordLink).toHaveBeenCalledTimes(1)
            expect(gone.backend.discordLink).toHaveBeenCalledTimes(1)
        })
    })
})
