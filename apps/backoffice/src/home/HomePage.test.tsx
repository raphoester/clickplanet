// @vitest-environment jsdom
import {Duration} from "@bufbuild/protobuf"
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import {afterEach, describe, expect, it, vi} from "vitest"
import {ACCOUNT, FakeAdmin} from "../fakeAdmin.ts"
import {OpenPlayer} from "../navigation.ts"
import {HomePage} from "./HomePage.tsx"

afterEach(cleanup)

function show(admin: FakeAdmin, onGo = vi.fn(), openPlayer = vi.fn()) {
    render(
        <OpenPlayer.Provider value={openPlayer}>
            <HomePage clients={admin.clients()} onGo={onGo}/>
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
    it("says who plays now, what they did in the last day, and what waits for the operator", async () => {
        const admin = new FakeAdmin()
        show(admin)

        expect(await screen.findByText(/^Last 24 hours:/)).toBeTruthy()
        expect(within(tile("Playing now")).getByText("2")).toBeTruthy()
        expect(within(tile("Playing now")).getByText("1 signed in")).toBeTruthy()
        expect(within(tile("Map tabs open")).getByText("38")).toBeTruthy()
        expect(within(tile("Map tabs open")).getByText("25 networks or accounts")).toBeTruthy()
        expect(within(tile("Clicks")).getByText("1,296")).toBeTruthy()
        expect(within(tile("Players")).getByText("42")).toBeTruthy()
        expect(within(tile("Open reports")).getByText("1")).toBeTruthy()
        expect(within(tile("Notes today")).getByText("2")).toBeTruthy()

        expect(admin.askedFor("getActivity")).toEqual([{span: new Duration({seconds: 86400n})}])
        expect(admin.askedFor("listReports")).toEqual([{openOnly: true, limit: 500}])
    })

    it("reads another span when one is picked", async () => {
        const admin = new FakeAdmin()
        show(admin)
        await screen.findByText(/^Last 24 hours:/)

        fireEvent.click(screen.getByRole("button", {name: "6 hours"}))

        expect(await screen.findByText(/^Last 6 hours:/)).toBeTruthy()
        expect(admin.askedFor("getActivity")[1]).toEqual({span: new Duration({seconds: 21600n})})
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

    it("says where the map changed hands the most", async () => {
        show(new FakeAdmin())

        expect(await screen.findByText("Busiest: 45°N 2°E: 30 changes on 4 tiles")).toBeTruthy()
    })

    it("lists the countries most held first, and by what they won or lost on demand", async () => {
        show(new FakeAdmin())
        const countries = await screen.findByRole("region", {name: "Countries"})
        const firstCountry = () => within(countries).getAllByRole("row")[1].textContent ?? ""

        expect(firstCountry()).toContain("France")
        fireEvent.click(within(countries).getByRole("button", {name: "Lost"}))
        expect(firstCountry()).toContain("Algeria")
    })

    it("still says who plays when the activity cannot be read", async () => {
        const admin = new FakeAdmin()
        admin.refuse("getActivity", Code.Unavailable)
        show(admin)

        expect(await screen.findByText("The call failed.")).toBeTruthy()
        expect(within(tile("Playing now")).getByText("2")).toBeTruthy()
    })
})
