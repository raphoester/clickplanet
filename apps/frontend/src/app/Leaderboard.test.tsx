// @vitest-environment jsdom
import {afterEach, describe, expect, it} from "vitest"
import {cleanup, render, screen, within} from "@testing-library/react"
import Leaderboard from "./Leaderboard.tsx"
import {Countries} from "../domain/countries.ts"

const entry = (code: string, tiles: number) => ({country: Countries.get(code)!, tiles})

const rows = () => screen.queryAllByRole("row").slice(1) // drop the header row
const cells = () => rows().map(r => within(r).getAllByRole("cell").map(c => c.textContent))

afterEach(cleanup)

describe("Leaderboard", () => {
    it("renders one row per country, in the order it was given", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 500), entry("jp", 250)]}/>)

        expect(cells()).toEqual([
            ["1", "🇫🇷 France", "500", "50.00"],
            ["2", "🇯🇵 Japan", "250", "25.00"],
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

    /** The UK nations' flags are long enough to have eaten the whole budget. */
    it("does not chop a country name to fit its flag", () => {
        render(<Leaderboard tilesCount={100} data={[entry("gb-eng", 5)]}/>)
        expect(screen.getByText("🏴󠁧󠁢󠁥󠁮󠁧󠁿 England")).toBeDefined()
    })

    /**
     * The table used to sit under a "Hide" button and nothing else, so the only
     * thing naming what would disappear was the button that hid it.
     */
    it("names itself, and labels its columns in words", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 500)]}/>)

        expect(screen.getByRole("region", {name: "Leaderboard"})).toBeDefined()
        expect(screen.getAllByRole("columnheader").map(h => h.textContent))
            .toEqual(["#", "Country", "Tiles", "Share"])
    })

    /** So a player can find themselves without counting down the table. */
    it("marks the player's own row", () => {
        render(<Leaderboard tilesCount={1000}
                            data={[entry("fr", 500), entry("jp", 250)]}
                            highlight={Countries.get("jp")!}/>)

        const marked = rows().filter(r => r.getAttribute("aria-current") === "true")
        expect(marked).toHaveLength(1)
        expect(within(marked[0]).getByText("🇯🇵 Japan")).toBeDefined()
    })

    it("marks nothing when the player's country holds no tile", () => {
        render(<Leaderboard tilesCount={1000}
                            data={[entry("fr", 500)]}
                            highlight={Countries.get("jp")!}/>)

        expect(rows().filter(r => r.getAttribute("aria-current") === "true")).toEqual([])
    })

    /** Collapsing is the card's job now; this component has no control of its own. */
    it("owns no toggle of its own", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 500)]}/>)
        expect(screen.queryAllByRole("button")).toEqual([])
    })
})
