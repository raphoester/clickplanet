import {describe, expect, it} from "vitest"
import {Season, SeasonTotals} from "../backends/season.ts"
import {Round, Score} from "../backends/standings.ts"
import {battleFlagRowsOf, bigNumbersOf, seasonFlagRowsOf, slidesOf} from "./seasonStats.ts"

const SEASON: Season = {
    number: 0, startsAt: Date.UTC(2026, 9, 8), endsAt: Date.UTC(2026, 9, 31, 23), bookId: "season-0", finaleRoundId: "season-0/finale",
}
const score = (rank: number, countryCode: string, points: number): Score => ({rank, countryCode, points, roundsWon: 0})
const TOTALS: SeasonTotals = {takes: 900, strikes: 40, spreads: 3, encloses: 2, bombs: 12, shields: 30, fortifies: 5, players: 70}
const FINALE: Round = {
    roundId: "season-0/finale", times: 3, startedAt: 0, endedAt: 1,
    standings: [
        {rank: 1, countryCode: "de", share: 0.3, points: 75},
        {rank: 2, countryCode: "fr", share: 0.2, points: 54},
        {rank: 11, countryCode: "jp", share: 0.01, points: 0},
    ],
    takes: [{countryCode: "fr", tiles: 400}, {countryCode: "de", tiles: 350}],
    players: [], before: [], after: [],
}

describe("slidesOf", () => {
    it("shows the Final Battle's flags and players between the season's and the numbers", () => {
        expect(slidesOf(SEASON)).toEqual(["flags", "players", "battleFlags", "battlePlayers", "numbers"])
    })

    it("skips the Final Battle for a season that had none", () => {
        expect(slidesOf({...SEASON, finaleRoundId: undefined})).toEqual(["flags", "players", "numbers"])
    })
})

describe("seasonFlagRowsOf", () => {
    it("is the table's first ten, each with its bar against the first", () => {
        const table = Array.from({length: 12}, (_, i) => score(i + 1, `c${i}`, 120 - i * 10))
        const rows = seasonFlagRowsOf(table)

        expect(rows).toHaveLength(10)
        expect(rows[0].width).toBe(1)
        expect(rows[1].width).toBeCloseTo(110 / 120)
    })
})

describe("battleFlagRowsOf", () => {
    it("is the flags that scored in the battle, with the tiles their players took", () => {
        expect(battleFlagRowsOf(FINALE).map((row) => [row.countryCode, row.points, row.tiles]))
            .toEqual([["de", 75, 350], ["fr", 54, 400]])
    })
})

describe("bigNumbersOf", () => {
    it("tells the ledger's totals, the season's days, the flags that scored and the Final Battle's clicks", () => {
        expect(bigNumbersOf(SEASON, TOTALS, [score(1, "de", 3), score(2, "fr", 1)], FINALE)).toEqual([
            {label: "Clicks", value: 900},
            {label: "Players", value: 70},
            {label: "Days", value: 24},
            {label: "Bombs dropped", value: 12},
            {label: "Shields placed", value: 30},
            {label: "Shield hits", value: 40},
            {label: "Lands fortified", value: 5},
            {label: "Flags that scored", value: 2},
            {label: "Final Battle clicks", value: 750},
        ])
    })

    it("leaves out what was not read", () => {
        expect(bigNumbersOf(SEASON, undefined, undefined, undefined)).toEqual([{label: "Days", value: 24}])
    })
})
