// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import {Season, SeasonTotals} from "../../backends/season.ts"
import {Round, Score, Standing} from "../../backends/standings.ts"
import {NameColor} from "../../backends/player.ts"
import HallOfFame, {HallOfFameButton} from "./HallOfFame.tsx"

const score = (rank: number, countryCode: string, points: number): Score => ({rank, countryCode, points, roundsWon: 0})
const player = (rank: number, name: string, countryCode: string, tiles: number): Standing =>
    ({rank, accountId: `account-${name}`, name, color: NameColor.UNSPECIFIED, countryCode, tiles})

const SEASON_ZERO: Season = {
    number: 0, startsAt: Date.UTC(2026, 9, 8), endsAt: Date.UTC(2026, 9, 31, 20), bookId: "season-0", finaleRoundId: "season-0/finale",
}
const SEASON_ONE: Season = {number: 1, startsAt: Date.UTC(2026, 10, 1, 15), endsAt: Date.UTC(2026, 10, 29, 20), bookId: "season-1"}
const SEASON_TWO: Season = {number: 2, startsAt: Date.UTC(2026, 11, 1, 15), endsAt: Date.UTC(2026, 11, 29, 20), bookId: "season-2"}

const FINALE: Round = {
    roundId: "season-0/finale", times: 3, startedAt: 0, endedAt: 1,
    standings: [{rank: 1, countryCode: "fr", share: 0.3, points: 75}],
    takes: [{countryCode: "fr", tiles: 4200}],
    players: [player(1, "Ana", "fr", 640)],
    before: [], after: [],
}
const TOTALS: SeasonTotals = {takes: 1_284_310, strikes: 1, spreads: 1, encloses: 1, bombs: 9_864, shields: 22_530, fortifies: 3, players: 2_416}

const books = (tables: Record<string, Score[]>) => ({
    scores: async (bookId: string) => tables[bookId] ?? [],
    board: async () => [player(1, "Nova", "de", 9000), player(2, "Kai", "jp", 7000)],
    round: async (roundId: string) => roundId === FINALE.roundId ? FINALE : undefined,
    bookHistory: async () => [{roundId: "season-0/day-1", endedAt: Date.UTC(2026, 9, 8, 21), results: [], table: [score(1, "de", 25)]}],
})
const gamemaster = {seasonTotals: async () => TOTALS}

const plaques = () => [...document.querySelectorAll(".hall-of-fame-plaque")] as HTMLElement[]
const steps = () => [...document.querySelectorAll(".podium-step")].map((step) => step.textContent)

afterEach(cleanup)

describe("HallOfFame", () => {
    it("shows each ended season, oldest first, then the season that runs, with their dates", () => {
        render(<HallOfFame teased={SEASON_TWO} ended={[SEASON_ZERO, SEASON_ONE]} standings={books({})} turn={gamemaster} onClose={vi.fn()}/>)

        expect(plaques().map((plaque) => within(plaque).getByText(/^Season \d$/).textContent))
            .toEqual(["Season 0", "Season 1", "Season 2"])
        expect(within(plaques()[0]).getByText("8 Oct – 31 Oct 2026")).toBeDefined()
        expect(within(plaques()[2]).getByText("?")).toBeDefined()
        expect(within(plaques()[2]).queryByRole("button", {name: "View"})).toBeNull()
        expect(within(plaques()[0]).getByRole("link", {name: "Map"}).getAttribute("href")).toBe("/archive")
    })

    it("shows each ended season's winner by its flag, every country level at the top", async () => {
        render(<HallOfFame ended={[SEASON_ZERO, SEASON_ONE]}
                           standings={books({"season-0": [score(1, "de", 365), score(2, "fr", 354)], "season-1": [score(1, "fr", 300), score(1, "jp", 300)]})}
                           turn={gamemaster}
                           onClose={vi.fn()}/>)

        expect(await within(plaques()[0]).findByRole("img", {name: "Germany"})).toBeDefined()
        expect(await within(plaques()[1]).findByRole("img", {name: "France and Japan"})).toBeDefined()
    })

    it("opens View on the podium, 2nd, 1st and 3rd, then every country of the table", async () => {
        render(<HallOfFame ended={[SEASON_ZERO]}
                           standings={books({"season-0": [score(1, "de", 365), score(2, "fr", 354), score(3, "jp", 300), score(4, "br", 10)]})}
                           turn={gamemaster}
                           onClose={vi.fn()}/>)

        fireEvent.click(screen.getByRole("button", {name: "View"}))

        await screen.findAllByText("Brazil")
        expect(steps()).toEqual(["France2", "Germany1", "Japan3"])
        expect(document.querySelectorAll(".season-stats-ranking .season-stats-row")).toHaveLength(4)
    })

    it("says so when nobody scored in an ended season", async () => {
        render(<HallOfFame ended={[SEASON_ZERO]} standings={books({})} turn={gamemaster} onClose={vi.fn()}/>)

        expect(await screen.findByText("No winner")).toBeDefined()
    })

    it("opens an ended season's slides on View, and goes back to the plaques", async () => {
        render(<HallOfFame teased={SEASON_ONE} ended={[SEASON_ZERO]} standings={books({"season-0": [score(1, "de", 365)]})} turn={gamemaster}
                           onClose={vi.fn()}/>)

        fireEvent.click(screen.getByRole("button", {name: "View"}))

        expect(screen.getByRole("dialog", {name: "Season 0"})).toBeDefined()
        expect(screen.getAllByRole("button", {name: /^Slide \d of 6$/})).toHaveLength(6)
        expect(await screen.findByText("Nova")).toBeDefined()
        expect(await screen.findByText("Ana")).toBeDefined()
        expect(await screen.findByText("4,200 tiles")).toBeDefined()
        expect(await screen.findByText("1,284,310")).toBeDefined()

        fireEvent.click(screen.getByRole("button", {name: "Hall of Fame"}))
        expect(screen.getByRole("dialog", {name: "Hall of Fame"})).toBeDefined()
    })
})

describe("HallOfFameButton", () => {
    it("opens the Hall of Fame, and Escape closes it", () => {
        render(<HallOfFameButton teased={SEASON_ONE} ended={[SEASON_ZERO]} standings={books({})} turn={gamemaster}/>)
        expect(screen.queryByRole("dialog")).toBeNull()

        fireEvent.click(screen.getByRole("button", {name: "Hall of Fame"}))
        expect(screen.getByRole("dialog", {name: "Hall of Fame"})).toBeDefined()

        fireEvent.keyDown(document, {key: "Escape"})
        expect(screen.queryByRole("dialog")).toBeNull()
    })
})
