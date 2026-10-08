import {describe, expect, it} from "vitest"
import {Race, RoundStanding, Score} from "../backends/standings.ts"
import {Countries} from "./countries.ts"
import {LeaderboardEntry} from "./leaderboard.ts"
import {countryLines} from "./race.ts"

const today = (rank: number, countryCode: string, points: number): RoundStanding => ({rank, countryCode, share: 0.1, points})
const score = (rank: number, countryCode: string, points: number, roundsWon = 0): Score => ({rank, countryCode, points, roundsWon})
const race = (number: number, standings: RoundStanding[], scores: Score[]): Race =>
    ({round: {number, endsAt: 0, finale: false, standings}, scores})
const held = (code: string, tiles: number): LeaderboardEntry => ({country: Countries.get(code)!, tiles})
const shown = (lines: ReturnType<typeof countryLines>) =>
    lines.map((line) => `${line.country.code} ${line.tiles} ${line.points}+${line.today}`)

const MAP = [held("de", 900), held("fr", 700), held("es", 300), held("bg", 50)]
const RACE = race(5, [today(1, "de", 25), today(2, "fr", 18), today(3, "es", 15)], [score(1, "fr", 43, 1), score(2, "it", 30, 1)])

describe("countryLines", () => {
    it("keeps the map's order by territory, with each country's points beside its tiles", () => {
        expect(shown(countryLines(MAP, RACE, "territory"))).toEqual([
            "de 900 0+25",
            "fr 700 43+18",
            "es 300 0+15",
            "bg 50 0+0",
        ])
    })

    it("orders by the season: the points, with what each would score if the day ended now", () => {
        expect(shown(countryLines(MAP, RACE, "season"))).toEqual([
            "fr 700 43+18",
            "it 0 30+0",
            "de 900 0+25",
            "es 300 0+15",
            "bg 50 0+0",
        ])
    })

    it("lists a country that holds no ground in the season for the points it has", () => {
        expect(shown(countryLines([], race(1, [], [score(1, "it", 30)]), "season"))).toEqual(["it 0 30+0"])
        expect(countryLines([], race(1, [], [score(1, "it", 30)]), "territory")).toEqual([])
    })

    it("orders the countries level on points by their tiles", () => {
        expect(shown(countryLines([held("es", 10), held("pt", 80)], race(1, [], []), "season"))).toEqual([
            "pt 80 0+0",
            "es 10 0+0",
        ])
    })

    it("reads no race as no points", () => {
        expect(shown(countryLines(MAP, undefined, "season"))).toEqual(["de 900 0+0", "fr 700 0+0", "es 300 0+0", "bg 50 0+0"])
    })
})
