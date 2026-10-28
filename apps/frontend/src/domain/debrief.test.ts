import {describe, expect, it} from "vitest"
import {Round} from "../backends/standings.ts"
import {Debrief} from "../backends/turn.ts"
import {closedOf, debriefRoundOf, endedBookOf, numbersOf} from "./debrief.ts"

const ENDED_AT = Date.UTC(2026, 9, 31, 23)

const DAY: Debrief = {kind: "day", season: 0, number: 24, roundId: "season-0/day-24", endedAt: Date.UTC(2026, 9, 31, 21)}
const BATTLE: Debrief = {kind: "battle", season: 0, number: 2, entryId: "finale", times: 3, endsSeason: true, roundId: "finale", endedAt: ENDED_AT}
const SEASON: Debrief = {kind: "season", season: 0, bookId: "season-0", endedAt: ENDED_AT, finaleRoundId: "finale", finaleBattle: BATTLE}

const ROUND: Round = {
    roundId: "finale",
    times: 3,
    startedAt: Date.UTC(2026, 9, 31, 21),
    endedAt: ENDED_AT,
    standings: [{rank: 1, countryCode: "de", share: 0.3, points: 75}],
    takes: [{countryCode: "de", tiles: 900}],
    players: [],
    before: [{rank: 1, countryCode: "fr", points: 300, roundsWon: 9}],
    after: [{rank: 1, countryCode: "de", points: 365, roundsWon: 8}],
}

describe("numbersOf", () => {
    it("reads a day's or a battle's round, the season's finale, or the book's scores with no finale", () => {
        expect(numbersOf(DAY)).toEqual({kind: "round", roundId: "season-0/day-24"})
        expect(numbersOf(BATTLE)).toEqual({kind: "round", roundId: "finale"})
        expect(numbersOf(SEASON)).toEqual({kind: "round", roundId: "finale"})
        expect(numbersOf({...SEASON, finaleRoundId: ""})).toEqual({kind: "scores", bookId: "season-0"})
    })
})

describe("closedOf", () => {
    it("names the moment in the gamemaster's words, with scoring's numbers", () => {
        expect(closedOf(BATTLE, ROUND)).toEqual({
            moment: "battle", season: 0, number: 2, roundId: "finale", endedAt: ENDED_AT, finale: true,
            standings: ROUND.standings, before: ROUND.before, after: ROUND.after, times: 3, startedAt: ROUND.startedAt,
            entryId: "finale", takes: ROUND.takes, players: [],
        })
        expect(closedOf(DAY, {...ROUND, roundId: "season-0/day-24", times: 1})).toMatchObject({moment: "day", number: 24, finale: false, entryId: ""})
        expect(closedOf(SEASON, ROUND)).toMatchObject({moment: "season", season: 0, number: 2, entryId: "", finale: true, after: ROUND.after})
        expect(closedOf({...SEASON, finaleBattle: undefined}, ROUND)).toMatchObject({moment: "season", number: 0, entryId: ""})
    })

    it("ends a season no round finished on the book's scores", () => {
        const scores = [{rank: 1, countryCode: "de", points: 365, roundsWon: 8}]
        expect(closedOf({...SEASON, finaleRoundId: "", finaleBattle: undefined}, undefined, scores)).toMatchObject({before: [], after: scores, roundId: ""})
    })
})

describe("debriefRoundOf and endedBookOf", () => {
    it("are the round a debrief tells of, and the book a season's closed", () => {
        expect([DAY, BATTLE, SEASON, undefined].map(debriefRoundOf)).toEqual(["season-0/day-24", "finale", "finale", ""])
        expect([DAY, BATTLE, SEASON, undefined].map(endedBookOf)).toEqual([undefined, undefined, "season-0", undefined])
    })
})
