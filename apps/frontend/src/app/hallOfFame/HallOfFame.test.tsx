// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import {sampleEndedSeason} from "../../backends/fakeHallOfFameBackend.ts"
import {EndedSeason} from "../../backends/hallOfFame.ts"
import {NameColor} from "../../backends/player.ts"
import HallOfFame from "./HallOfFame.tsx"
import HallOfFameButton from "./HallOfFameButton.tsx"

const SAMPLE_ENDED_SEASON = sampleEndedSeason()

afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
})

const rowsOf = (table: HTMLElement) => within(table).getAllByRole("row").slice(1)
    .map((row) => row.textContent)

describe("HallOfFame", () => {
    it("crowns the champion of each season, newest first", () => {
        const earlier: EndedSeason = {...SAMPLE_ENDED_SEASON, number: 0, scores: [{rank: 1, countryCode: "de", points: 90, roundsWon: 3}]}
        const latest: EndedSeason = {...SAMPLE_ENDED_SEASON, number: 1}
        render(<HallOfFame hall={[latest, earlier]} countryCode="fr" onClose={vi.fn()}/>)

        const dialog = screen.getByRole("dialog", {name: "Hall of Fame"})
        expect(within(dialog).getAllByRole("region").map((season) => season.querySelector(".menu-label")?.textContent))
            .toEqual(["Season 1", "Season 0"])
        expect(within(screen.getByRole("region", {name: "Season 1"})).getByText("France", {selector: ".hall-champion-name"}))
            .toBeDefined()
        expect(within(screen.getByRole("region", {name: "Season 0"})).getByText("Germany", {selector: ".hall-champion-name"}))
            .toBeDefined()
    })

    it("crowns every country level on first place", () => {
        const level: EndedSeason = {
            ...SAMPLE_ENDED_SEASON,
            scores: [{rank: 1, countryCode: "de", points: 100, roundsWon: 2}, {rank: 1, countryCode: "fr", points: 100, roundsWon: 2}],
        }
        render(<HallOfFame hall={[level]} countryCode="fr" onClose={vi.fn()}/>)

        expect([...document.querySelectorAll(".hall-champion-name")].map((name) => name.textContent)).toEqual(["Germany", "France"])
    })

    it("shows the top of the final table with its points, and the player's country under it", () => {
        render(<HallOfFame hall={[SAMPLE_ENDED_SEASON]} countryCode="us" onClose={vi.fn()}/>)

        const rows = rowsOf(screen.getByRole("table", {name: "Countries, season 0"}))
        expect(rows).toHaveLength(11)
        expect(rows[0]).toBe("1France214")
        expect(rows[9]).toBe("10UAE12")
        expect(rows[10]).toBe("12USA3")
        expect(screen.getByRole("table", {name: "Countries, season 0"}).querySelector("[aria-current]")?.textContent)
            .toBe("12USA3")
    })

    it("ranks the season's players by their tiles", () => {
        render(<HallOfFame hall={[SAMPLE_ENDED_SEASON]} countryCode="fr" onClose={vi.fn()}/>)

        const rows = rowsOf(screen.getByRole("table", {name: "Players, season 0"}))
        expect(rows).toHaveLength(10)
        expect(rows[0]).toBe("1AnaOG9840")
        expect(rows[5]).toBe("5Ольга3990")
    })

    it("marks the player's own row", () => {
        render(<HallOfFame hall={[SAMPLE_ENDED_SEASON]}
                           countryCode="fr"
                           accountId={SAMPLE_ENDED_SEASON.standings[1].accountId}
                           onClose={vi.fn()}/>)

        expect(screen.getByRole("table", {name: "Players, season 0"}).querySelector("[aria-current]")?.textContent)
            .toBe("2Mateus8112")
    })

    it("opens a player's card by its account", () => {
        const onOpenPlayer = vi.fn()
        render(<HallOfFame hall={[SAMPLE_ENDED_SEASON]} countryCode="fr" onOpenPlayer={onOpenPlayer} onClose={vi.fn()}/>)

        fireEvent.click(screen.getByRole("button", {name: "Mateus"}))

        expect(onOpenPlayer).toHaveBeenCalledWith(expect.objectContaining({
            accountId: SAMPLE_ENDED_SEASON.standings[1].accountId, name: "Mateus", countryCode: "br", color: NameColor.TEAL,
        }))
    })

    it("closes", () => {
        const onClose = vi.fn()
        render(<HallOfFame hall={[SAMPLE_ENDED_SEASON]} countryCode="fr" onClose={onClose}/>)

        fireEvent.click(screen.getByRole("button", {name: "Close"}))

        expect(onClose).toHaveBeenCalled()
    })
})

describe("HallOfFameButton", () => {
    it("opens the Hall of Fame", () => {
        const onOpen = vi.fn()
        render(<HallOfFameButton compact={false} besideSeason onOpen={onOpen}/>)

        fireEvent.click(screen.getByRole("button", {name: "Hall of Fame"}))

        expect(onOpen).toHaveBeenCalled()
    })
})
