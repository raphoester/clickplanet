import {describe, expect, it} from "vitest"
import {EndedSeason} from "../backends/hallOfFame.ts"
import {Score} from "../backends/standings.ts"
import {championsOf, finalTableOf, HALL_TABLE_ROWS, holds} from "./hallOfFame.ts"

const score = (rank: number, countryCode: string, points: number): Score => ({rank, countryCode, points, roundsWon: 0})

const season = (number: number, scores: Score[]): EndedSeason => ({number, endedAt: 0, scores, standings: []})

describe("the champions", () => {
    it("are the countries first in the final table", () => {
        expect(championsOf(season(0, [score(1, "fr", 200), score(2, "de", 150)]))).toEqual([score(1, "fr", 200)])
    })

    it("are every country level on first place", () => {
        const level = season(0, [score(1, "de", 100), score(1, "fr", 100), score(3, "it", 15)])

        expect(championsOf(level).map((champion) => champion.countryCode)).toEqual(["de", "fr"])
    })

    it("are nobody when no country scored", () => {
        expect(championsOf(season(0, []))).toEqual([])
    })
})

describe("the final table shown", () => {
    const scores = Array.from({length: 14}, (_, i) => score(i + 1, `c${i + 1}`, 100 - i))

    it("lists the top countries, and the player's under them when it is not one", () => {
        const table = finalTableOf(season(0, scores), "c13")

        expect(table.listed).toHaveLength(HALL_TABLE_ROWS)
        expect(table.listed[0].countryCode).toBe("c1")
        expect(table.own).toEqual(scores[12])
    })

    it("adds nothing when the player's country is in the top", () => {
        expect(finalTableOf(season(0, scores), "c3").own).toBeUndefined()
    })

    it("adds nothing when the player's country did not score", () => {
        expect(finalTableOf(season(0, scores), "zz").own).toBeUndefined()
    })
})

describe("whether the hall holds a season", () => {
    it("is whether a season of that number is in it", () => {
        const hall = [season(1, []), season(0, [])]

        expect(holds(hall, 0)).toBe(true)
        expect(holds(hall, 2)).toBe(false)
        expect(holds([], 0)).toBe(false)
    })
})
