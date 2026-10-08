import {describe, expect, it} from "vitest"
import {ClosedRound, RoundStanding, Score} from "../backends/standings.ts"
import {medalOf, moveOf, podiumOf, REVEAL_WITHIN_MS, revealDue, revealKeyOf, revealRowsOf, roundOverLine} from "./roundReveal.ts"

const result = (rank: number, countryCode: string, points: number): RoundStanding => ({rank, countryCode, share: 0.1, points})
const score = (rank: number, countryCode: string, points: number): Score => ({rank, countryCode, points, roundsWon: 0})

const ENDED_AT = Date.UTC(2026, 9, 16, 21)
const DAY: ClosedRound = {
    season: 0,
    number: 5,
    endedAt: ENDED_AT,
    finale: false,
    standings: [result(1, "fr", 25), result(2, "de", 18), result(3, "es", 15), result(4, "it", 12)],
    before: [score(1, "de", 50), score(2, "br", 40), score(3, "fr", 30)],
    after: [score(1, "de", 68), score(2, "fr", 55), score(3, "br", 40), score(4, "es", 15), score(5, "it", 12)],
}

describe("revealDue", () => {
    it("plays a round closed in the last hours that this browser has not seen", () => {
        expect(revealDue(DAY, ENDED_AT + 60_000, null)).toBe(true)
        expect(revealDue(DAY, ENDED_AT + REVEAL_WITHIN_MS - 1, "0:4:1")).toBe(true)
    })

    it("does not play it twice", () => {
        expect(revealDue(DAY, ENDED_AT + 60_000, revealKeyOf(DAY))).toBe(false)
    })

    it("does not play a round closed long ago, or no round at all", () => {
        expect(revealDue(DAY, ENDED_AT + REVEAL_WITHIN_MS, null)).toBe(false)
        expect(revealDue(undefined, ENDED_AT, null)).toBe(false)
    })
})

describe("podiumOf", () => {
    it("is the three countries that placed best", () => {
        expect(podiumOf(DAY).map((standing) => standing.countryCode)).toEqual(["fr", "de", "es"])
    })

    it("keeps a tie on the podium and never shows more than three", () => {
        const tied = {...DAY, standings: [result(1, "fr", 25), result(1, "de", 25), result(3, "es", 15), result(3, "it", 15)]}

        expect(podiumOf(tied).map((standing) => `${standing.rank} ${standing.countryCode}`)).toEqual(["1 fr", "1 de", "3 es"])
    })

    it("is empty when nobody scored", () => {
        expect(podiumOf({...DAY, standings: []})).toEqual([])
    })
})

describe("revealRowsOf", () => {
    const rows = revealRowsOf(DAY, "fr")

    it("ends in the season's order after the round, with the points each country gained", () => {
        expect(rows.map((row) => `${row.to} ${row.countryCode} ${row.after.points} +${row.gained}`)).toEqual([
            "0 de 68 +18",
            "1 fr 55 +25",
            "2 br 40 +0",
            "3 es 15 +15",
            "4 it 12 +12",
        ])
    })

    it("starts in the season's order before the round, with the countries new to the table below", () => {
        expect([...rows].sort((a, b) => a.from - b.from).map((row) => row.countryCode)).toEqual(["de", "br", "fr", "es", "it"])
    })

    it("marks the player's country", () => {
        expect(rows.filter((row) => row.mine).map((row) => row.countryCode)).toEqual(["fr"])
    })

    it("cuts the table and adds the player's country under it", () => {
        expect(revealRowsOf(DAY, "it", 2).map((row) => `${row.to} ${row.countryCode}`)).toEqual(["0 de", "1 fr", "2 it"])
    })

    it("says how each country moved", () => {
        expect(rows.map(moveOf)).toEqual([
            {kind: "same"},
            {kind: "up", by: 1},
            {kind: "down", by: 1},
            {kind: "new"},
            {kind: "new"},
        ])
    })
})

describe("roundOverLine", () => {
    it("names the day, or the Final Battle", () => {
        expect(roundOverLine({number: 5, finale: false})).toBe("Day 5 is over!")
        expect(roundOverLine({number: 23, finale: true})).toBe("The Final Battle is over!")
    })
})

describe("medalOf", () => {
    it("is a medal for the podium and nothing below it", () => {
        expect([1, 2, 3, 4].map(medalOf)).toEqual(["🥇", "🥈", "🥉", ""])
    })
})
