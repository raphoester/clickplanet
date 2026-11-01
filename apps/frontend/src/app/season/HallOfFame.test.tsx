// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import {Season} from "../../backends/season.ts"
import {Score} from "../../backends/standings.ts"
import HallOfFame, {HallOfFameButton} from "./HallOfFame.tsx"

const score = (rank: number, countryCode: string, points: number): Score => ({rank, countryCode, points, roundsWon: 0})

const SEASON_ZERO: Season = {number: 0, startsAt: Date.UTC(2026, 9, 8), endsAt: Date.UTC(2026, 9, 31, 20), bookId: "season-0"}
const SEASON_ONE: Season = {number: 1, startsAt: Date.UTC(2026, 10, 1, 15), endsAt: Date.UTC(2026, 10, 29, 20), bookId: "season-1"}
const SEASON_TWO: Season = {number: 2, startsAt: Date.UTC(2026, 11, 1, 15), endsAt: Date.UTC(2026, 11, 29, 20), bookId: "season-2"}

const books = (tables: Record<string, Score[]>) => ({scores: async (bookId: string) => tables[bookId] ?? []})

const plaques = () => [...document.querySelectorAll(".hall-of-fame-plaque")] as HTMLElement[]
const steps = (plaque: HTMLElement) => [...plaque.querySelectorAll(".hall-of-fame-step")].map((step) => step.textContent)

afterEach(cleanup)

describe("HallOfFame", () => {
    it("shows each ended season, oldest first, then the season that runs, with their dates", () => {
        render(<HallOfFame teased={SEASON_TWO} ended={[SEASON_ZERO, SEASON_ONE]} backend={books({})} onClose={vi.fn()}/>)

        expect(plaques().map((plaque) => within(plaque).getByText(/^Season \d$/).textContent))
            .toEqual(["Season 0", "Season 1", "Season 2"])
        expect(within(plaques()[0]).getByText("8 Oct – 31 Oct 2026")).toBeDefined()
        expect(within(plaques()[2]).getByText("?")).toBeDefined()
    })

    it("stands each ended season's top three on a podium, 2nd, 1st and 3rd", async () => {
        render(<HallOfFame ended={[SEASON_ZERO]}
                           backend={books({"season-0": [score(1, "de", 365), score(2, "fr", 354), score(3, "jp", 300), score(4, "br", 10)]})}
                           onClose={vi.fn()}/>)

        await screen.findByText("Germany")
        expect(steps(plaques()[0])).toEqual(["France2", "Germany1", "Japan3"])
    })

    it("keeps countries level on a step of the same rank", async () => {
        render(<HallOfFame ended={[SEASON_ZERO]}
                           backend={books({"season-0": [score(1, "de", 365), score(1, "fr", 365), score(3, "jp", 300)]})}
                           onClose={vi.fn()}/>)

        await screen.findByText("Germany")
        expect(screen.getAllByLabelText("Rank 1")).toHaveLength(2)
    })

    it("says so when nobody scored in an ended season", async () => {
        render(<HallOfFame ended={[SEASON_ZERO]} backend={books({})} onClose={vi.fn()}/>)

        expect(await screen.findByText("No winner")).toBeDefined()
    })
})

describe("HallOfFameButton", () => {
    it("opens the Hall of Fame, and Escape closes it", () => {
        render(<HallOfFameButton teased={SEASON_ONE} ended={[SEASON_ZERO]} backend={books({})}/>)
        expect(screen.queryByRole("dialog")).toBeNull()

        fireEvent.click(screen.getByRole("button", {name: "Hall of Fame"}))
        expect(screen.getByRole("dialog", {name: "Hall of Fame"})).toBeDefined()

        fireEvent.keyDown(document, {key: "Escape"})
        expect(screen.queryByRole("dialog")).toBeNull()
    })
})
