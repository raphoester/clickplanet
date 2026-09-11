// @vitest-environment jsdom
import {afterEach, describe, expect, it} from "vitest"
import {cleanup, render, screen, within} from "@testing-library/react"
import Leaderboard from "./Leaderboard.tsx"
import {Countries} from "../domain/countries.ts"
import {TileDelta, TileDeltas} from "../domain/tileDeltas.ts"

const entry = (code: string, tiles: number) => ({country: Countries.get(code)!, tiles})

const deltas = (pairs: Record<string, number>): TileDeltas => new Map(
    Object.entries(pairs).map(([code, net]): [string, TileDelta] =>
        [code, {net, beat: 1, at: 0}]))

const rows = () => screen.queryAllByRole("row").slice(1)
const cells = () => rows().map(r => within(r).getAllByRole("cell").map(c => c.textContent))

afterEach(cleanup)

describe("Leaderboard", () => {
    it("renders one row per country, in the order it was given", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 500), entry("jp", 250)]}/>)

        expect(cells()).toEqual([
            ["1", "France", "500", "50.00"],
            ["2", "Japan", "250", "25.00"],
        ])
    })

    it("shows each country's share of the map to two decimals", () => {
        render(<Leaderboard tilesCount={257_948} data={[entry("fr", 1)]}/>)
        expect(screen.getByText("0.00")).toBeDefined()
    })

    it("renders nothing but the header when no country holds a tile", () => {
        render(<Leaderboard tilesCount={1000} data={[]}/>)
        expect(rows()).toEqual([])
    })

    it("draws a country's flag from the atlas, never as an emoji", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 500)]}/>)

        const row = rows()[0]
        expect(row.querySelectorAll(".country-flag")).toHaveLength(1)
        expect(row.textContent).not.toMatch(/\p{RI}|\p{Extended_Pictographic}/u)
    })

    it("does not chop a country name to fit its flag", () => {
        render(<Leaderboard tilesCount={100} data={[entry("gb-eng", 5)]}/>)
        expect(screen.getByText("England")).toBeDefined()
    })

    it("names itself, and labels its columns in words", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 500)]}/>)

        expect(screen.getByRole("region", {name: "Leaderboard"})).toBeDefined()
        expect(screen.getAllByRole("columnheader").map(h => h.textContent))
            .toEqual(["#", "Country", "Tiles", "Share"])
    })

    it("marks the player's own row", () => {
        render(<Leaderboard tilesCount={1000}
                            data={[entry("fr", 500), entry("jp", 250)]}
                            highlight={Countries.get("jp")!}/>)

        const marked = rows().filter(r => r.getAttribute("aria-current") === "true")
        expect(marked).toHaveLength(1)
        expect(within(marked[0]).getByText("Japan")).toBeDefined()
    })

    it("marks nothing when the player's country holds no tile", () => {
        render(<Leaderboard tilesCount={1000}
                            data={[entry("fr", 500)]}
                            highlight={Countries.get("jp")!}/>)

        expect(rows().filter(r => r.getAttribute("aria-current") === "true")).toEqual([])
    })

    it("owns no toggle of its own", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 500)]}/>)
        expect(screen.queryAllByRole("button")).toEqual([])
    })
})

describe("Leaderboard tile deltas", () => {
    const badges = () => rows().map(r => r.querySelector(".leaderboard-delta")?.textContent)

    it("floats what a country just won next to its count", () => {
        render(<Leaderboard tilesCount={1000}
                            data={[entry("fr", 503), entry("jp", 250)]}
                            deltas={deltas({fr: 3})}/>)

        expect(badges()).toEqual(["+3", undefined])
    })

    it("spells a loss with its minus", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 498)]} deltas={deltas({fr: -2})}/>)
        expect(badges()).toEqual(["-2"])
    })

    it("badges nothing while the board is still", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 500)]}/>)
        expect(badges()).toEqual([undefined])
    })

    it("colours the count itself the way the badge reads", () => {
        render(<Leaderboard tilesCount={1000}
                            data={[entry("fr", 503), entry("jp", 248), entry("gb-eng", 10)]}
                            deltas={deltas({fr: 3, jp: -2})}/>)

        expect(rows().map(r => r.querySelector(".leaderboard-table-tiles")!.className))
            .toEqual([
                expect.stringContaining("leaderboard-tiles-up"),
                expect.stringContaining("leaderboard-tiles-down"),
                expect.not.stringContaining("leaderboard-tiles-"),
            ])
    })

    it("keeps the badge out of the row a screen reader reads", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 503)]} deltas={deltas({fr: 3})}/>)

        const badge = rows()[0].querySelector(".leaderboard-delta")!
        expect(badge.getAttribute("aria-hidden")).toBe("true")
    })

    it("leaves the count itself as the number it is", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 503)]} deltas={deltas({fr: 3})}/>)
        expect(cells()).toEqual([["1", "France", "503+3", "50.30"]])
    })
})
