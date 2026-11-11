import {describe, expect, it} from "vitest"
import {Race, Round, RoundStanding, Score} from "../backends/standings.ts"
import {Countries} from "./countries.ts"
import {LeaderboardEntry} from "./leaderboard.ts"
import {countryLines, raceOf, scoringNow, shownOrder} from "./race.ts"

const today = (rank: number, countryCode: string, points: number): RoundStanding => ({rank, countryCode, share: 0.1, points})
const score = (rank: number, countryCode: string, points: number, roundsWon = 0): Score => ({rank, countryCode, points, roundsWon})
const open = (roundId: string, standings: RoundStanding[], times = 1): Round =>
    ({roundId, times, startedAt: 0, endedAt: 0, standings, takes: [], players: [], before: [], after: [], start: []})
const race = (number: number, standings: RoundStanding[], scores: Score[]): Race =>
    ({today: open(`season-0/day-${number}`, standings), scores})
const held = (code: string, tiles: number): LeaderboardEntry => ({country: Countries.get(code)!, tiles})
const shown = (lines: ReturnType<typeof countryLines>) =>
    lines.map((line) => `${line.country.code} ${line.tiles} ${line.points}+${line.today}`)

const MAP = [held("de", 900), held("fr", 700), held("es", 300), held("bg", 50)]
const RACE = race(5, [today(1, "de", 25), today(2, "fr", 18), today(3, "es", 15)], [score(1, "fr", 43, 1), score(2, "it", 30, 1)])

describe("shownOrder", () => {
    it("is the order picked while a race is on, and the points when none was picked", () => {
        expect(shownOrder(RACE, "tiles")).toBe("tiles")
        expect(shownOrder(RACE, "gain")).toBe("gain")
        expect(shownOrder(RACE, undefined)).toBe("points")
    })

    it("is the tiles with no race, whatever was picked", () => {
        expect(shownOrder(undefined, "points")).toBe("tiles")
        expect(shownOrder(undefined, "gain")).toBe("tiles")
    })

    it("is the points for the gain and for today while no round runs", () => {
        expect(shownOrder({scores: []}, "gain")).toBe("points")
        expect(shownOrder({scores: []}, "today")).toBe("points")
        expect(shownOrder(RACE, "today")).toBe("today")
    })
})

describe("countryLines", () => {
    it("keeps the map's order by tiles, with each country's points beside its tiles", () => {
        expect(shown(countryLines(MAP, RACE, "tiles"))).toEqual([
            "de 900 0+25",
            "fr 700 43+18",
            "es 300 0+15",
            "bg 50 0+0",
        ])
    })

    it("orders by the points won on the days that ended", () => {
        expect(shown(countryLines(MAP, RACE, "points"))).toEqual([
            "fr 700 43+18",
            "it 0 30+0",
            "de 900 0+25",
            "es 300 0+15",
            "bg 50 0+0",
        ])
    })

    it("lists a country that holds no ground by points for the points it has", () => {
        expect(shown(countryLines([], race(1, [], [score(1, "it", 30)]), "points"))).toEqual(["it 0 30+0"])
        expect(countryLines([], race(1, [], [score(1, "it", 30)]), "tiles")).toEqual([])
    })

    it("orders the countries level on points by their tiles, not by what they score today", () => {
        const day = race(1, [today(9, "es", 2), today(10, "it", 1)], [])

        expect(shown(countryLines([held("es", 3_817), held("it", 4_867)], day, "points"))).toEqual([
            "it 4867 0+1",
            "es 3817 0+2",
        ])
    })

    it("orders by the tiles gained since the round opened, then by the tiles held", () => {
        const day: Race = {...RACE, today: {...RACE.today!, start: [held("de", 950), held("fr", 600), held("bg", 50)]
            .map((entry) => ({countryCode: entry.country.code, tiles: entry.tiles}))}}

        expect(countryLines(MAP, day, "gain").map((line) => `${line.country.code} ${line.gain}`)).toEqual([
            "es 300",
            "fr 100",
            "bg 0",
            "it 0",
            "de -50",
        ])
    })

    it("reads no race as no points", () => {
        expect(shown(countryLines(MAP, undefined, "points"))).toEqual(["de 900 0+0", "fr 700 0+0", "es 300 0+0", "bg 50 0+0"])
    })

    // Day 11 of season 1, as the board showed it: Bulgaria 7th on its tiles and 3rd on its gain won the day.
    const DAY_11: [code: string, tiles: number, gain: number][] = [
        ["br", 57_249, 61], ["ie", 53_790, -1_902], ["se", 33_402, -612], ["pw", 27_252, 524], ["fr", 25_327, -1_222],
        ["de", 18_807, -8_075], ["bg", 8_741, 2_593], ["at", 5_950, 769], ["hu", 5_747, 645], ["pl", 5_412, 2_009],
        ["ro", 4_599, 2_809], ["co", 3_049, -434], ["is", 3_010, 3_010], ["id", 2_465, 349], ["va", 2_160, -48],
        ["in", 1_433, 365], ["nl", 1_089, -346], ["ru", 367, 55], ["cy", 278, 38], ["ps", 122, -160], ["bt", 100, 79],
        ["gb", 41, -359], ["us", 31, 31], ["ca", 18, -2], ["pr", 14, 14], ["it", 9, -87], ["tr", 9, -5],
    ]
    const DAY_11_PLACES = ["bg", "pw", "br", "ro", "at", "is", "pl", "hu", "id", "in"]
    const day11: Race = {
        scores: [],
        today: {
            ...open("season-1/day-11", DAY_11_PLACES.map((code, i) => today(i + 1, code, [25, 18, 15, 12, 10, 8, 6, 4, 2, 1][i]))),
            start: DAY_11.map(([countryCode, tiles, gain]) => ({countryCode, tiles: tiles - gain})),
        },
    }
    const day11Map = DAY_11.map(([code, tiles]) => held(code, tiles))

    it("ranks each country on its tiles and on its gain, among the countries that hold ground", () => {
        const ranks = new Map(countryLines(day11Map, day11, "tiles").map((line) => [line.country.code, line.ranks]))

        expect(ranks.get("bg")).toEqual({tiles: 7, gain: 3})
        expect(ranks.get("pw")).toEqual({tiles: 4, gain: 7})
        expect(ranks.get("br")).toEqual({tiles: 1, gain: 11})
        expect(ranks.get("fr")).toEqual({tiles: 5, gain: 25})
        expect(ranks.get("it")).toEqual({tiles: 26, gain: 19})
        expect(ranks.get("tr")).toEqual({tiles: 26, gain: 17})
    })

    it("orders by the day's places, which the sum of the two ranks makes, the gain's rank breaking a tie", () => {
        const lines = countryLines(day11Map, day11, "today")
        const bySum = [...lines].sort((a, b) =>
            a.ranks!.tiles + a.ranks!.gain - (b.ranks!.tiles + b.ranks!.gain) || a.ranks!.gain - b.ranks!.gain)

        expect(lines.slice(0, 10).map((line) => `${line.country.code} +${line.today}`)).toEqual([
            "bg +25", "pw +18", "br +15", "ro +12", "at +10", "is +8", "pl +6", "hu +4", "id +2", "in +1",
        ])
        expect(bySum.slice(0, 10).map((line) => line.country.code)).toEqual(DAY_11_PLACES)
    })

    it("lists the countries the round did not place after the others, by their tiles", () => {
        expect(countryLines(day11Map, day11, "today").slice(10, 13).map((line) => line.country.code)).toEqual(["ie", "se", "fr"])
    })

    it("has no ranks while no round runs", () => {
        expect(countryLines(MAP, {scores: []}, "points").map((line) => line.ranks)).toEqual([undefined, undefined, undefined, undefined])
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
