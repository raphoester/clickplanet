// @vitest-environment jsdom
import {afterEach, describe, expect, it} from "vitest"
import {cleanup, render, screen, within} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
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
            ["1.", "🇫🇷 France", "500", "50.00"],
            ["2.", "🇯🇵 Japan", "250", "25.00"],
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

    it("collapses and expands the table", async () => {
        const user = userEvent.setup()
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 500)]}/>)

        expect(rows()).toHaveLength(1)

        await user.click(screen.getByRole("button", {name: "Hide"}))
        expect(screen.queryAllByRole("row")).toEqual([])

        await user.click(screen.getByRole("button", {name: "Leaderboard"}))
        expect(rows()).toHaveLength(1)
    })
})
