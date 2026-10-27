import {describe, expect, it} from "vitest"
import {Season} from "../backends/season.ts"
import {ClosedRound, Score} from "../backends/standings.ts"
import {championNames, championOf, reigningChampionOf, winsLine} from "./champion.ts"

const score = (rank: number, countryCode: string, points: number): Score => ({rank, countryCode, points, roundsWon: 0})

const FINALE: ClosedRound = {
    season: 0,
    number: 23,
    endedAt: Date.UTC(2026, 9, 31, 23),
    finale: true,
    standings: [],
    before: [score(1, "fr", 300), score(2, "de", 290)],
    after: [score(1, "de", 365), score(2, "fr", 354)],
}

const SEASON_ZERO: Season = {number: 0, lastDayEndsAt: Date.UTC(2026, 9, 31, 21), endsAt: Date.UTC(2026, 9, 31, 23)}
const SEASON_ONE: Season = {number: 1, lastDayEndsAt: Date.UTC(2026, 10, 30, 21), endsAt: Date.UTC(2026, 10, 30, 23)}

describe("championOf", () => {
    it("is the country first in the table once the Final Battle closed", () => {
        expect(championOf(FINALE)).toEqual({season: 0, countryCodes: ["de"]})
    })

    it("is every country level at the top", () => {
        expect(championOf({...FINALE, after: [score(1, "de", 365), score(1, "fr", 365)]}))
            .toEqual({season: 0, countryCodes: ["de", "fr"]})
    })

    it("is nobody after a day, or a season nobody scored in", () => {
        expect(championOf({...FINALE, finale: false})).toBeUndefined()
        expect(championOf({...FINALE, after: []})).toBeUndefined()
        expect(championOf(undefined)).toBeUndefined()
    })
})

describe("reigningChampionOf", () => {
    it("reigns while no newer season is on", () => {
        expect(reigningChampionOf(undefined, FINALE)).toEqual({season: 0, countryCodes: ["de"]})
        expect(reigningChampionOf(SEASON_ZERO, FINALE)).toEqual({season: 0, countryCodes: ["de"]})
    })

    it("steps down once the next season is on", () => {
        expect(reigningChampionOf(SEASON_ONE, FINALE)).toBeUndefined()
    })
})

describe("winsLine", () => {
    it("names the champion and its season", () => {
        expect(winsLine({season: 0, countryCodes: ["de"]})).toBe("Germany wins Season 0")
    })

    it("names champions together", () => {
        expect(winsLine({season: 2, countryCodes: ["de", "fr", "es"]})).toBe("Germany, France and Spain win Season 2")
        expect(championNames({season: 2, countryCodes: ["de", "fr"]})).toBe("Germany and France")
    })
})
