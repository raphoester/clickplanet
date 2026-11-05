// @vitest-environment jsdom
import {act, cleanup, fireEvent, render, screen, waitFor} from "@testing-library/react"
import {afterEach, describe, expect, it} from "vitest"
import {hashOf, routeOf} from "./navigation.ts"
import {useRoute} from "./useRoute.ts"

afterEach(() => {
    cleanup()
    window.location.hash = ""
})

function Shown() {
    const [route, go] = useRoute()
    return (
        <>
            <p>{route.item === "" ? route.page : `${route.page} ${route.item}`}</p>
            <button type="button" onClick={() => go({page: "replay", item: ""})}>Replay</button>
            <button type="button" onClick={() => go({page: "players", item: "2001:db8::/64"})}>Address</button>
        </>
    )
}

describe("routeOf", () => {
    it("reads the page an address names, and what is open on it", () => {
        expect(routeOf("#/reports")).toEqual({page: "reports", item: ""})
        expect(routeOf("#players")).toEqual({page: "players", item: ""})
        expect(routeOf("#/reports/report-3")).toEqual({page: "reports", item: "report-3"})
    })

    it("opens on the first page for an address that names none", () => {
        expect(routeOf("")).toEqual({page: "home", item: ""})
        expect(routeOf("#/nowhere/report-3")).toEqual({page: "home", item: ""})
    })

    it("reads back an address with a slash in it, as a /64 has", () => {
        const route = {page: "players" as const, item: "2001:db8:1:2::/64"}
        expect(routeOf(hashOf(route))).toEqual(route)
    })
})

describe("useRoute", () => {
    it("opens on the route in the address bar, as after a reload", () => {
        window.location.hash = "#/reports/report-3"
        render(<Shown/>)
        expect(screen.getByText("reports report-3")).toBeTruthy()
    })

    it("puts the route picked in the address bar", () => {
        render(<Shown/>)
        fireEvent.click(screen.getByRole("button", {name: "Replay"}))
        expect(screen.getByText("replay")).toBeTruthy()
        expect(window.location.hash).toBe("#/replay")
        fireEvent.click(screen.getByRole("button", {name: "Address"}))
        expect(screen.getByText("players 2001:db8::/64")).toBeTruthy()
    })

    it("follows the address bar, as Back does", async () => {
        render(<Shown/>)
        act(() => {
            window.location.hash = "#/titles"
        })
        await waitFor(() => expect(screen.getByText("titles")).toBeTruthy())
    })
})
