import {describe, expect, it} from "vitest"
import {Race, RoundStanding, Score} from "../backends/standings.ts"
import {raceTable} from "./race.ts"

const today = (rank: number, countryCode: string, points: number): RoundStanding => ({rank, countryCode, share: 0.1, points})
const score = (rank: number, countryCode: string, points: number, roundsWon = 0): Score => ({rank, countryCode, points, roundsWon})
const race = (standings: RoundStanding[], scores: Score[]): Race =>
    ({round: {number: 3, endsAt: 0, finale: false, standings}, scores})

describe("raceTable", () => {
    it("ranks the countries by their points with what they would score if the day ended now", () => {
        const table = raceTable(race(
            [today(1, "de", 25), today(2, "fr", 18), today(3, "es", 15)],
            [score(1, "fr", 43, 1), score(2, "de", 25, 1)],
        ), "fr")

        expect(table).toEqual({
            listed: [
                {rank: 1, countryCode: "fr", points: 43, today: 18},
                {rank: 2, countryCode: "de", points: 25, today: 25},
                {rank: 3, countryCode: "es", points: 0, today: 15},
            ],
        })
    })

    it("lists a country that scores nothing today for the points it has", () => {
        const table = raceTable(race([today(1, "de", 25)], [score(1, "it", 30)]), "fr")

        expect(table.listed).toEqual([
            {rank: 1, countryCode: "it", points: 30, today: 0},
            {rank: 2, countryCode: "de", points: 0, today: 25},
        ])
    })

    it("leaves out a country that has no points and scores nothing today", () => {
        const table = raceTable(race([today(1, "de", 25), today(11, "fr", 0)], []), "fr")

        expect(table.listed.map((row) => row.countryCode)).toEqual(["de"])
        expect(table.below).toBeUndefined()
    })

    it("shares a rank between countries level on the total, the better day first", () => {
        const table = raceTable(race([today(1, "es", 25), today(2, "fr", 18)], [score(1, "fr", 7)]), "fr")

        expect(table.listed).toEqual([
            {rank: 1, countryCode: "es", points: 0, today: 25},
            {rank: 1, countryCode: "fr", points: 7, today: 18},
        ])
    })

    it("shows the top ten, and the country played for under them when it is further down", () => {
        const scores = Array.from({length: 12}, (_, i) => score(i + 1, `c${String(i).padStart(2, "0")}`, 100 - i))

        const table = raceTable(race([], scores), "c11")

        expect(table.listed).toHaveLength(10)
        expect(table.listed.at(-1)?.countryCode).toBe("c09")
        expect(table.below).toEqual({rank: 12, countryCode: "c11", points: 89, today: 0})
    })

    it("reads a race with no round in progress", () => {
        expect(raceTable({scores: [score(1, "fr", 75, 1)]}, "fr").listed)
            .toEqual([{rank: 1, countryCode: "fr", points: 75, today: 0}])
    })
})
