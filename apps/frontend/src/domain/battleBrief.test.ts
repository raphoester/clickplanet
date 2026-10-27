import {describe, expect, it} from "vitest"
import {NameColor} from "../backends/player.ts"
import {ClosedRound, Race, Standing} from "../backends/standings.ts"
import {Battle} from "./battle.ts"
import {briefAwaited, briefDescriptionOf, briefStagesOf, takesRowsOf} from "./battleBrief.ts"

const STARTED_AT = Date.UTC(2026, 9, 31, 21)
const ENDED_AT = Date.UTC(2026, 9, 31, 23)

const player = (rank: number, name: string, countryCode: string, tiles: number): Standing =>
    ({rank, accountId: `account-${name}`, name, color: NameColor.UNSPECIFIED, countryCode, tiles})

const BATTLE_ROUND: ClosedRound = {
    season: 0,
    number: 1,
    endedAt: ENDED_AT,
    finale: true,
    standings: [{rank: 1, countryCode: "de", share: 0.3, points: 75}],
    before: [],
    after: [],
    own: true,
    times: 3,
    startedAt: STARTED_AT,
    takes: [{countryCode: "de", tiles: 4000}, {countryCode: "fr", tiles: 2000}, {countryCode: "es", tiles: 1000}],
    players: [player(1, "Ana", "de", 900)],
}

const battle: Battle = {since: STARTED_AT, endsAt: ENDED_AT, description: "Triple points."}

describe("briefStagesOf", () => {
    it("shows who won, the tiles each flag took, then the top players", () => {
        expect(briefStagesOf(BATTLE_ROUND)).toEqual(["podium", "takes", "players"])
    })

    it("skips a stage with nothing in it, and keeps the head when nothing is", () => {
        expect(briefStagesOf({...BATTLE_ROUND, standings: []})).toEqual(["takes", "players"])
        expect(briefStagesOf({...BATTLE_ROUND, takes: [{countryCode: "de", tiles: 0}], players: []})).toEqual(["podium"])
        expect(briefStagesOf({...BATTLE_ROUND, standings: [], takes: [], players: []})).toEqual(["podium"])
    })
})

describe("takesRowsOf", () => {
    it("draws each flag's tiles against the flag that took the most, and marks the player's", () => {
        expect(takesRowsOf(BATTLE_ROUND, "fr")).toEqual([
            {countryCode: "de", tiles: 4000, width: 1, mine: false},
            {countryCode: "fr", tiles: 2000, width: 0.5, mine: true},
            {countryCode: "es", tiles: 1000, width: 0.25, mine: false},
        ])
    })

    it("cuts the list and keeps the player's flag under it", () => {
        expect(takesRowsOf(BATTLE_ROUND, "es", 1).map((row) => row.countryCode)).toEqual(["de", "es"])
    })
})

describe("briefDescriptionOf", () => {
    it("is the words of the battle the page saw play, when it is this round's", () => {
        expect(briefDescriptionOf(battle, BATTLE_ROUND)).toBe("Triple points.")
    })

    it("is the battle's words when the round was cut short of the battle's end", () => {
        expect(briefDescriptionOf({...battle, endsAt: ENDED_AT + 3_600_000}, BATTLE_ROUND)).toBe("Triple points.")
    })

    it("is none for another battle, or when the page saw none", () => {
        expect(briefDescriptionOf({since: STARTED_AT - 86_400_000, endsAt: ENDED_AT - 86_400_000, description: "Rush."}, BATTLE_ROUND)).toBe("")
        expect(briefDescriptionOf(undefined, BATTLE_ROUND)).toBe("")
    })
})

describe("briefAwaited", () => {
    const racing = (round: Race["round"]): Race => ({round, scores: []})

    it("waits while the race still shows the battle's round of its own", () => {
        expect(briefAwaited(battle, racing({number: 1, endsAt: ENDED_AT, finale: true, standings: [], own: true, times: 3}))).toBe(true)
    })

    it("waits for nothing once the round closed, or when the battle had no round of its own", () => {
        expect(briefAwaited(battle, racing(undefined))).toBe(false)
        expect(briefAwaited(battle, racing({number: 6, endsAt: ENDED_AT + 86_400_000, finale: false, standings: [], own: false, times: 1}))).toBe(false)
        expect(briefAwaited(battle, racing({number: 2, endsAt: ENDED_AT + 3_600_000, finale: false, standings: [], own: true, times: 2}))).toBe(false)
        expect(briefAwaited(battle, undefined)).toBe(false)
    })
})
