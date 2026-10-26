// @vitest-environment jsdom
import {Timestamp} from "@bufbuild/protobuf"
import {Code} from "@connectrpc/connect"
import {act, cleanup, fireEvent, render, screen, waitFor, within} from "@testing-library/react"
import {afterEach, describe, expect, it, onTestFinished, vi} from "vitest"
import {ACCOUNT, AT, FakeAdmin} from "../fakeAdmin.ts"
import {GetLedgerResponse, LedgerChange} from "../gen/grpc/planet/v1/admin_pb.ts"
import {TileUpdate} from "../gen/grpc/planet/v1/planet_pb.ts"
import {OpenPlayer} from "../navigation.ts"
import {HomePage, REFRESH_MS} from "./HomePage.tsx"
import {Places} from "./places.ts"

afterEach(cleanup)

const MINUTE = 60_000

function shownAs(state: DocumentVisibilityState) {
    Object.defineProperty(document, "visibilityState", {value: state, configurable: true})
    act(() => {
        document.dispatchEvent(new Event("visibilitychange"))
    })
}

function windowsAsked(admin: FakeAdmin): string[][] {
    const asked = admin.askedFor("getLedger") as {since: {toDate(): Date}, until: {toDate(): Date}}[]
    return asked.map(request => [request.since.toDate().toISOString(), request.until.toDate().toISOString()])
}

// The fake ledger's four tiles, all in one degree by Paris.
const PARIS: Places = {
    latitudes: Float32Array.from([48.8, 48.6, 48.7, 48.5]),
    longitudes: Float32Array.from([2.3, 2.1, 2.6, 2.4]),
}

function show(admin: FakeAdmin, places = () => Promise.resolve(PARIS)) {
    const onGo = vi.fn()
    const openPlayer = vi.fn()
    render(
        <OpenPlayer.Provider value={openPlayer}>
            <HomePage clients={admin.clients()} onGo={onGo} places={places}/>
        </OpenPlayer.Provider>,
    )
    return {onGo, openPlayer}
}

function tile(label: string): HTMLElement {
    const found = screen.getByText(label, {selector: ".tile-label"}).closest("li")
    if (found === null) throw new Error(`no tile ${label}`)
    return found
}

describe("the home page", () => {
    it("counts the clicks and the players of the span from the ledger, beside who plays and what waits", async () => {
        vi.useFakeTimers({now: AT, toFake: ["Date"]})
        onTestFinished(() => {
            vi.useRealTimers()
        })
        const admin = new FakeAdmin()
        show(admin)

        expect(await screen.findByText(/^Last 24 hours:/)).toBeTruthy()
        expect(within(tile("Clicks")).getByText("4")).toBeTruthy()
        expect(within(tile("Players")).getByText("2")).toBeTruthy()
        expect(within(tile("Playing now")).getByText("1 signed in")).toBeTruthy()
        expect(within(tile("Open reports")).getByText("1")).toBeTruthy()
        expect(within(tile("Notes today")).getByText("2")).toBeTruthy()
    })

    it("asks the ledger for whole slots back from now to now, and again for the span picked", async () => {
        vi.useFakeTimers({now: new Date("2026-10-23T12:07:00Z"), toFake: ["Date"]})
        try {
            const admin = new FakeAdmin()
            show(admin)
            await screen.findByText(/^Last 24 hours:/)

            fireEvent.click(screen.getByRole("button", {name: "6 hours"}))
            await screen.findByText(/^Last 6 hours:/)

            expect(windowsAsked(admin)).toEqual([
                ["2026-10-22T12:15:00.000Z", "2026-10-23T12:07:00.000Z"],
                ["2026-10-23T06:10:00.000Z", "2026-10-23T12:07:00.000Z"],
            ])
        } finally {
            vi.useRealTimers()
        }
    })

    it("reads again only from a minute before its last read, and counts what it read before", async () => {
        vi.useFakeTimers({now: AT, toFake: ["Date"]})
        onTestFinished(() => {
            vi.useRealTimers()
        })
        const admin = new FakeAdmin()
        show(admin)
        const reload = screen.getByRole("button", {name: "Reload"})
        await waitFor(() => expect(reload).toHaveProperty("disabled", false))
        expect(within(tile("Clicks")).getByText("4")).toBeTruthy()

        vi.setSystemTime(new Date(AT.getTime() + 2 * MINUTE))
        admin.ledger = new GetLedgerResponse({
            changes: [new LedgerChange({
                at: Timestamp.fromDate(new Date(AT.getTime() + MINUTE)), by: "scope:198.51.100.4",
                update: new TileUpdate({tileId: 4, countryId: "dz", previousCountryId: "fr", clicked: true}),
            })],
        })
        fireEvent.click(reload)

        await waitFor(() => expect(within(tile("Clicks")).getByText("5")).toBeTruthy())
        expect(within(tile("Players")).getByText("3")).toBeTruthy()
        expect(windowsAsked(admin)[1]).toEqual(["2026-10-18T11:59:00.000Z", "2026-10-18T12:02:00.000Z"])
    })

    it("reads every page of the ledger, one after the other", async () => {
        vi.useFakeTimers({now: AT, toFake: ["Date"]})
        onTestFinished(() => {
            vi.useRealTimers()
        })
        const admin = new FakeAdmin()
        admin.ledgerPages = [[new LedgerChange({
            at: Timestamp.fromDate(new Date(AT.getTime() - 30 * MINUTE)), by: "scope:198.51.100.4",
            update: new TileUpdate({tileId: 4, countryId: "dz", previousCountryId: "fr", clicked: true}),
        })]]
        show(admin)

        await waitFor(() => expect(within(tile("Clicks")).getByText("5")).toBeTruthy())
        expect((admin.askedFor("getLedger") as {page: string}[]).map(request => request.page)).toEqual(["", "1"])
    })

    it("reads again a minute after its last read ended, never while one runs nor while the page is hidden", async () => {
        vi.useFakeTimers({now: AT, toFake: ["Date", "setTimeout", "clearTimeout"]})
        onTestFinished(() => {
            vi.useRealTimers()
            shownAs("visible")
        })
        const admin = new FakeAdmin()
        const release = admin.hold("getLedger")
        show(admin)
        const reads = () => admin.askedFor("getLedger").length

        await act(() => vi.advanceTimersByTimeAsync(5 * REFRESH_MS))
        expect(reads()).toBe(1)
        expect(screen.getByRole("button", {name: "Reload"})).toHaveProperty("disabled", true)

        await act(async () => {
            release()
            await vi.advanceTimersByTimeAsync(0)
        })
        expect(screen.getByRole("button", {name: "Reload"})).toHaveProperty("disabled", false)
        await act(() => vi.advanceTimersByTimeAsync(REFRESH_MS - 1))
        expect(reads()).toBe(1)
        await act(() => vi.advanceTimersByTimeAsync(1))
        expect(reads()).toBe(2)
        await act(() => vi.advanceTimersByTimeAsync(0))

        shownAs("hidden")
        await act(() => vi.advanceTimersByTimeAsync(5 * REFRESH_MS))
        expect(reads()).toBe(2)

        shownAs("visible")
        await act(() => vi.advanceTimersByTimeAsync(0))
        expect(reads()).toBe(3)
    })

    it("goes to the page of what waits for the operator", async () => {
        const {onGo} = show(new FakeAdmin())
        await screen.findByText(/^Last 24 hours:/)

        fireEvent.click(screen.getByRole("button", {name: /Open reports/}))
        fireEvent.click(screen.getByRole("button", {name: /Notes today/}))

        expect(onGo.mock.calls).toEqual([["reports"], ["notes"]])
    })

    it("opens a player playing now on the player page, and counts the guests without listing them", async () => {
        const {openPlayer} = show(new FakeAdmin())
        const playing = await screen.findByRole("region", {name: "Playing now"})

        expect(within(playing).getByText("1 signed in, 1 guest")).toBeTruthy()
        expect(within(playing).queryByText("guest_a1b2c3")).toBeNull()
        fireEvent.click(within(playing).getByRole("button", {name: /BraveFox42/}))

        expect(openPlayer).toHaveBeenCalledWith({caller: {accountId: ACCOUNT}, name: "BraveFox42"})
    })

    it("says where the map changed hands the most, from where the map puts each tile", async () => {
        show(new FakeAdmin())

        expect(await screen.findByText("Busiest: 48°N 2°E: 4 changes on 4 tiles")).toBeTruthy()
    })

    it("lists the countries most held first, and by what they won or lost on demand", async () => {
        show(new FakeAdmin())
        const countries = await screen.findByRole("region", {name: "Countries"})
        const firstCountry = () => within(countries).getAllByRole("row")[1].textContent ?? ""

        expect(firstCountry()).toContain("France")
        fireEvent.click(within(countries).getByRole("button", {name: "Lost"}))
        expect(firstCountry()).toContain("Algeria")
    })

    it("still shows the rest when the ledger or the map cannot be read", async () => {
        const admin = new FakeAdmin()
        admin.refuse("getLedger", Code.Unavailable)
        show(admin, () => Promise.reject(new Error("no map")))

        expect(await screen.findByText("The call failed.")).toBeTruthy()
        expect(within(tile("Playing now")).getByText("2")).toBeTruthy()
    })
})
