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

const leader = () => screen.queryByRole("region", {name: /^First: /})
const rows = () => screen.queryAllByRole("row").slice(1)
const cells = () => rows().map(r => within(r).getAllByRole("cell").map(c => c.textContent))

afterEach(cleanup)

describe("Leaderboard", () => {
    it("frames the first country, then lists the rest in the order it was given", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 500), entry("jp", 250), entry("de", 100)]}/>)

        expect(leader()!.getAttribute("aria-label")).toBe("First: France")
        expect(cells()).toEqual([
            ["2", "Japan", "250", "25.00"],
            ["3", "Germany", "100", "10.00"],
        ])
    })

    it("gives the first country its share of the map and its tiles", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 123)]}/>)

        expect(within(leader()!).getByText("12.30")).toBeDefined()
        expect(within(leader()!).getByText("123 tiles")).toBeDefined()
    })

    it("shows each country's share of the map to two decimals", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 500), entry("jp", 123)]}/>)
        expect(within(rows()[0]).getByText("12.30")).toBeDefined()
    })

    it("says a share is small rather than rounding it away to nothing", () => {
        render(<Leaderboard tilesCount={257_948} data={[entry("fr", 1)]}/>)
        expect(screen.getByText("<0.01")).toBeDefined()
    })

    it("still writes a plain zero for a country holding no tile at all", () => {
        render(<Leaderboard tilesCount={257_948} data={[entry("fr", 0)]}/>)
        expect(screen.getByText("0.00")).toBeDefined()
    })

    it("renders no frame and no table when no country holds a tile", () => {
        render(<Leaderboard tilesCount={1000} data={[]}/>)

        expect(leader()).toBeNull()
        expect(screen.queryByRole("table")).toBeNull()
    })

    it("draws a country's flag from the atlas, never as an emoji", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("jp", 600), entry("fr", 500)]}/>)

        for (const place of [leader()!, rows()[0]]) {
            expect(place.querySelectorAll(".country-flag")).toHaveLength(1)
            expect(place.textContent).not.toMatch(/\p{RI}|\p{Extended_Pictographic}/u)
        }
    })

    it("does not chop a country name to fit its flag", () => {
        render(<Leaderboard tilesCount={100} data={[entry("gb-eng", 5)]}/>)
        expect(screen.getByText("England")).toBeDefined()
    })

    it("names itself, and labels its columns in words", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 500), entry("jp", 250)]}/>)

        expect(screen.getByRole("region", {name: "Leaderboard"})).toBeDefined()
        expect(screen.getAllByRole("columnheader").map(h => h.textContent))
            .toEqual(["#", "Country", "Tiles", "% of map"])
    })

    it("marks the player's own row", () => {
        render(<Leaderboard tilesCount={1000}
                            data={[entry("fr", 500), entry("jp", 250)]}
                            highlight={Countries.get("jp")!}/>)

        const marked = rows().filter(r => r.getAttribute("aria-current") === "true")
        expect(marked).toHaveLength(1)
        expect(within(marked[0]).getByText("Japan")).toBeDefined()
        expect(leader()!.getAttribute("aria-current")).toBeNull()
    })

    it("marks the frame when the player's country leads", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 500), entry("jp", 250)]} highlight={Countries.get("fr")!}/>)

        expect(leader()!.getAttribute("aria-current")).toBe("true")
        expect(rows().filter(r => r.getAttribute("aria-current") === "true")).toEqual([])
    })

    it("marks nothing when the player's country holds no tile", () => {
        render(<Leaderboard tilesCount={1000}
                            data={[entry("fr", 500), entry("de", 100)]}
                            highlight={Countries.get("jp")!}/>)

        expect(leader()!.getAttribute("aria-current")).toBeNull()
        expect(rows().filter(r => r.getAttribute("aria-current") === "true")).toEqual([])
    })

    it("owns no toggle of its own", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 500), entry("jp", 250)]}/>)
        expect(screen.queryAllByRole("button")).toEqual([])
    })

    it("says how much slower the first country refills, off the toll", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 250)]} toll={[{share: 0.1, slowdown: 1.5}, {share: 0.2, slowdown: 2.5}]}/>)
        expect(within(leader()!).getByText("Refills 2.5× slower")).toBeDefined()
    })

    it("holds what it is given beside the first country", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 250)]} anthem={<p>the anthem</p>}/>)
        expect(within(leader()!).getByText("the anthem")).toBeDefined()
    })
})

describe("Leaderboard tile deltas", () => {
    const badges = () => [leader()!, ...rows()].map(r => r.querySelector(".leaderboard-delta")?.textContent)

    it("floats what a country just won next to its count", () => {
        render(<Leaderboard tilesCount={1000}
                            data={[entry("fr", 503), entry("jp", 250), entry("de", 100)]}
                            deltas={deltas({fr: 3, jp: 2})}/>)

        expect(badges()).toEqual(["+3", "+2", undefined])
    })

    it("spells a loss with its minus", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 600), entry("jp", 498)]} deltas={deltas({jp: -2})}/>)
        expect(badges()).toEqual([undefined, "-2"])
    })

    it("badges nothing while the board is still", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 500), entry("jp", 250)]}/>)
        expect(badges()).toEqual([undefined, undefined])
    })

    it("colours the count itself the way the badge reads", () => {
        render(<Leaderboard tilesCount={1000}
                            data={[entry("fr", 600), entry("jp", 503), entry("de", 248), entry("gb-eng", 10)]}
                            deltas={deltas({jp: 3, de: -2})}/>)

        expect(rows().map(r => r.querySelector(".leaderboard-table-tiles")!.className))
            .toEqual([
                expect.stringContaining("leaderboard-tiles-up"),
                expect.stringContaining("leaderboard-tiles-down"),
                expect.not.stringContaining("leaderboard-tiles-"),
            ])
    })

    it("keeps the badge out of what a screen reader reads", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 503), entry("jp", 250)]} deltas={deltas({fr: 3, jp: 1})}/>)

        for (const place of [leader()!, rows()[0]]) {
            expect(place.querySelector(".leaderboard-delta")!.getAttribute("aria-hidden")).toBe("true")
        }
    })

    it("leaves the count itself as the number it is", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 600), entry("jp", 503)]} deltas={deltas({jp: 3})}/>)
        expect(cells()).toEqual([["2", "Japan", "503+3", "50.30"]])
    })
})
