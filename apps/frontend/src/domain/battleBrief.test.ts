import {describe, expect, it} from "vitest"
import {NameColor} from "../backends/player.ts"
import {ClosedRound, Race, Standing} from "../backends/standings.ts"
import {Battle} from "./battle.ts"
import {briefAwaited, briefDescriptionOf, debriefedFor, debriefScreensOf, pointsRowsOf} from "./battleBrief.ts"

const STARTED_AT = Date.UTC(2026, 9, 31, 21)
const ENDED_AT = Date.UTC(2026, 9, 31, 23)

const player = (rank: number, name: string, countryCode: string, tiles: number): Standing =>
    ({rank, accountId: `account-${name}`, name, color: NameColor.UNSPECIFIED, countryCode, tiles})

const BATTLE_ROUND: ClosedRound = {
    season: 0,
    number: 1,
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
    own: true,
    times: 3,
    startedAt: STARTED_AT,
    entryId: "entry-final",
    takes: [],
    players: [player(1, "Ana", "de", 900)],
}

const battle: Battle = {entryId: "entry-final", since: STARTED_AT, endsAt: ENDED_AT, description: "Triple points."}
const another: Battle = {...battle, entryId: "entry-before", description: "Rush."}

describe("debriefedFor", () => {
    it("is the battle the page saw, by its entry's id, or any battle when the page saw none", () => {
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
    const racing = (round: Race["round"]): Race => ({round, scores: []})
    const own = {number: 1, endsAt: ENDED_AT, finale: true, standings: [], own: true, times: 3}

    it("waits while the race still shows the battle's round of its own", () => {
        expect(briefAwaited(battle, racing({...own, entryId: "entry-final"}))).toBe(true)
    })

    it("waits for nothing once the round closed, for a day, or for another entry's round", () => {
        expect(briefAwaited(battle, racing(undefined))).toBe(false)
        expect(briefAwaited(battle, racing({...own, own: false, finale: false, times: 1, entryId: ""}))).toBe(false)
        expect(briefAwaited(battle, racing({...own, entryId: "entry-next"}))).toBe(false)
        expect(briefAwaited(battle, undefined)).toBe(false)
    })
})
