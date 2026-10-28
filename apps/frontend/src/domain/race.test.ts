import {describe, expect, it} from "vitest"
import {Race, Round, RoundStanding, Score} from "../backends/standings.ts"
import {Countries} from "./countries.ts"
import {LeaderboardEntry} from "./leaderboard.ts"
import {countryLines, raceOf, scoringNow, shownOrder} from "./race.ts"

const today = (rank: number, countryCode: string, points: number): RoundStanding => ({rank, countryCode, share: 0.1, points})
const score = (rank: number, countryCode: string, points: number, roundsWon = 0): Score => ({rank, countryCode, points, roundsWon})
const open = (roundId: string, standings: RoundStanding[], times = 1): Round =>
    ({roundId, times, startedAt: 0, endedAt: 0, standings, takes: [], players: [], before: [], after: []})
const race = (number: number, standings: RoundStanding[], scores: Score[]): Race =>
    ({today: open(`season-0/day-${number}`, standings), scores})
const held = (code: string, tiles: number): LeaderboardEntry => ({country: Countries.get(code)!, tiles})
const shown = (lines: ReturnType<typeof countryLines>) =>
    lines.map((line) => `${line.country.code} ${line.tiles} ${line.points}+${line.today}`)

const MAP = [held("de", 900), held("fr", 700), held("es", 300), held("bg", 50)]
const RACE = race(5, [today(1, "de", 25), today(2, "fr", 18), today(3, "es", 15)], [score(1, "fr", 43, 1), score(2, "it", 30, 1)])

describe("shownOrder", () => {
    it("is the order picked while a race is on, and the season when none was picked", () => {
        expect(shownOrder(RACE, "territory")).toBe("territory")
        expect(shownOrder(RACE, undefined)).toBe("season")
    })

    it("is the territory with no race, whatever was picked", () => {
        expect(shownOrder(undefined, "season")).toBe("territory")
    })
})

describe("countryLines", () => {
    it("keeps the map's order by territory, with each country's points beside its tiles", () => {
        expect(shown(countryLines(MAP, RACE, "territory"))).toEqual([
            "de 900 0+25",
            "fr 700 43+18",
            "es 300 0+15",
            "bg 50 0+0",
        ])
    })

    it("orders by the season: the points won on the days that ended", () => {
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

    it("orders the countries level on points by their tiles, not by what they score today", () => {
        const day = race(1, [today(9, "es", 2), today(10, "it", 1)], [])

        expect(shown(countryLines([held("es", 3_817), held("it", 4_867)], day, "season"))).toEqual([
            "it 4867 0+1",
            "es 3817 0+2",
        ])
    })

    it("reads no race as no points", () => {
        expect(shown(countryLines(MAP, undefined, "season"))).toEqual(["de 900 0+0", "fr 700 0+0", "es 300 0+0", "bg 50 0+0"])
    })
})

describe("raceOf", () => {
    const day = open("season-0/day-5", [today(1, "de", 25)])
    const battle = open("entry-1", [today(1, "fr", 75)], 3)
    const rounds = new Map([[day.roundId, day], [battle.roundId, battle]])

    it("is today's round and the running battle's, as scoring sends them", () => {
        expect(raceOf([], rounds, "season-0/day-5", "entry-1")).toEqual({today: day, battle, scores: []})
        expect(raceOf([], rounds, "season-0/day-5", undefined)).toEqual({today: day, battle: undefined, scores: []})
    })

    it("drops a round once scored, and is nothing until the book's scores arrived", () => {
        const scored = new Map([[day.roundId, {...day, endedAt: 1}]])
        expect(raceOf([], scored, "season-0/day-5", undefined)?.today).toBeUndefined()
        expect(raceOf(undefined, rounds, "season-0/day-5", "entry-1")).toBeUndefined()
    })
})

describe("scoringNow", () => {
    it("is the running battle's round, else today's", () => {
        const day = open("season-0/day-5", [])
        const battle = open("entry-1", [], 3)
        expect(scoringNow({today: day, battle, scores: []})).toBe(battle)
        expect(scoringNow({today: day, scores: []})).toBe(day)
        expect(scoringNow(undefined)).toBeUndefined()
    })
})
