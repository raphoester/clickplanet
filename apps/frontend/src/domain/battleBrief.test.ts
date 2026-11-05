import {describe, expect, it} from "vitest"
import {NameColor} from "../backends/player.ts"
import {ClosedRound, Standing} from "../backends/standings.ts"
import {Debrief} from "../backends/turn.ts"
import {Battle} from "./battle.ts"
import {briefAwaited, briefDescriptionOf, dayBriefAwaited, debriefedFor, debriefScreensOf, pointsRowsOf} from "./battleBrief.ts"

const STARTED_AT = Date.UTC(2026, 9, 31, 21)
const ENDED_AT = Date.UTC(2026, 9, 31, 23)

const player = (rank: number, name: string, countryCode: string, tiles: number): Standing =>
    ({rank, accountId: `account-${name}`, name, color: NameColor.UNSPECIFIED, countryCode, tiles})

const BATTLE_ROUND: ClosedRound = {
    moment: "battle",
    season: 0,
    number: 1,
    roundId: "entry-final",
    endedAt: ENDED_AT,
    finale: true,
    standings: [
        {rank: 1, countryCode: "de", share: 0.3, points: 75},
        {rank: 2, countryCode: "fr", share: 0.2, points: 54},
        {rank: 3, countryCode: "es", share: 0.1, points: 45},
        {rank: 11, countryCode: "it", share: 0.01, points: 0},
    ],
    before: [],
    after: [],
    times: 3,
    startedAt: STARTED_AT,
    entryId: "entry-final",
    takes: [],
    players: [player(1, "Ana", "de", 900)],
}

const battle: Battle = {entryId: "entry-final", since: STARTED_AT, endsAt: ENDED_AT, description: "Triple points.", roundId: "entry-final"}
const another: Battle = {...battle, entryId: "entry-before", description: "Rush.", roundId: "entry-before"}

describe("debriefedFor", () => {
    it("is the battle the page saw, by its round, or any battle when the page saw none", () => {
        expect(debriefedFor(BATTLE_ROUND, battle)).toBe(true)
        expect(debriefedFor(BATTLE_ROUND, undefined)).toBe(true)
        expect(debriefedFor(BATTLE_ROUND, another)).toBe(false)
    })

    it("does not take a battle of the same window for this one", () => {
        expect(debriefedFor(BATTLE_ROUND, {...another, since: STARTED_AT, endsAt: ENDED_AT})).toBe(false)
    })
})

describe("debriefScreensOf", () => {
    it("is the battle finished, the points, then the top players", () => {
        expect(debriefScreensOf(BATTLE_ROUND)).toEqual(["over", "points", "players"])
    })

    it("skips the points or the players when there are none", () => {
        expect(debriefScreensOf({...BATTLE_ROUND, standings: []})).toEqual(["over", "players"])
        expect(debriefScreensOf({...BATTLE_ROUND, players: []})).toEqual(["over", "points"])
    })
})

describe("pointsRowsOf", () => {
    it("is each country that scored, its points before and after the times, drawn against the best", () => {
        expect(pointsRowsOf(BATTLE_ROUND, "fr")).toEqual([
            {rank: 1, countryCode: "de", share: 0.3, points: 75, won: 25, width: 1, mine: false},
            {rank: 2, countryCode: "fr", share: 0.2, points: 54, won: 18, width: 0.72, mine: true},
            {rank: 3, countryCode: "es", share: 0.1, points: 45, won: 15, width: 0.6, mine: false},
        ])
    })

    it("cuts the list and keeps the player's country under it", () => {
        expect(pointsRowsOf(BATTLE_ROUND, "es", 1).map((row) => row.countryCode)).toEqual(["de", "es"])
    })
})

describe("briefDescriptionOf", () => {
    it("is the words of the battle the page saw play", () => {
        expect(briefDescriptionOf(battle, BATTLE_ROUND)).toBe("Triple points.")
    })

    it("is none for another battle, or when the page saw none", () => {
        expect(briefDescriptionOf(another, BATTLE_ROUND)).toBe("")
        expect(briefDescriptionOf(undefined, BATTLE_ROUND)).toBe("")
    })
})

describe("briefAwaited", () => {
    const ended = (roundId: string, endsSeason = true): Debrief =>
        ({kind: "battle", season: 0, number: 1, entryId: roundId, times: 3, endsSeason, roundId, endedAt: ENDED_AT})
    const seasonEnded = (finaleRoundId: string): Debrief =>
        ({kind: "season", season: 0, bookId: "season-0", endedAt: ENDED_AT, finaleRoundId})

    it("waits while the gamemaster has not debriefed the battle's round", () => {
        expect(briefAwaited(battle, undefined)).toBe(true)
        expect(briefAwaited(battle, ended("entry-before", false))).toBe(true)
        expect(briefAwaited(battle, {kind: "day", season: 0, number: 24, roundId: "season-0/day-24", endedAt: ENDED_AT})).toBe(true)
    })

    it("waits for nothing once its round is debriefed, by the battle or the season, or for a battle with no round", () => {
        expect(briefAwaited(battle, ended("entry-final"))).toBe(false)
        expect(briefAwaited(battle, seasonEnded("entry-final"))).toBe(false)
        expect(briefAwaited({...battle, roundId: ""}, undefined)).toBe(false)
    })
})

describe("dayBriefAwaited", () => {
    const day = (endedAt: number): ClosedRound =>
        ({...BATTLE_ROUND, moment: "day", number: 24, roundId: "season-0/day-24", endedAt, finale: false, times: 1, entryId: ""})

    it("waits while the day the battle ended is not debriefed and read", () => {
        expect(dayBriefAwaited(battle, undefined)).toBe(true)
        expect(dayBriefAwaited(battle, day(STARTED_AT - 24 * 60 * 60 * 1000))).toBe(true)
    })

    it("waits for nothing once that day is read, or for a battle with no round", () => {
        expect(dayBriefAwaited(battle, day(STARTED_AT))).toBe(false)
        expect(dayBriefAwaited({...battle, roundId: ""}, undefined)).toBe(false)
    })
})
