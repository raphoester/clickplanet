import {describe, expect, it} from "vitest"
import {protoInt64, Timestamp} from "@bufbuild/protobuf"
import {Debrief, Timeline} from "../gen/grpc/gamemaster/v1/gamemaster_pb.ts"
import {BattleEnded, DayEnded, SeasonEnded} from "../gen/grpc/gamemaster/v1/events_pb.ts"
import {Round} from "../gen/grpc/scoring/v1/scoring_pb.ts"
import {dayOf, finaleAtOf} from "./race.ts"

const ENDED = 1_760_734_800_000

const dayFive = new Debrief({moment: {case: "day", value: new DayEnded({season: 1, day: 5, roundId: "season-1/day-5"})}})

function round(roundId: string): Round {
    return new Round({
        roundId, endedAtUnixMs: protoInt64.parse(ENDED),
        standings: [
            {rank: 3, countryId: "dz", share: 0.06, points: 15},
            {rank: 1, countryId: "fr", share: 0.12, points: 25},
            {rank: 2, countryId: "pw", share: 0.09, points: 18},
            {rank: 2, countryId: "br", share: 0.09, points: 18},
            {rank: 4, countryId: "ma", share: 0.05, points: 12},
        ],
        after: [{rank: 1, countryId: "fr", points: 86, roundsWon: 3}],
    })
}

describe("the round the gamemaster debriefed last", () => {
    it("puts the places that scored on its podium, ties sharing one, and keeps the season after it", () => {
        expect(dayOf(dayFive, round("season-1/day-5"))).toEqual({
            season: 1, number: 5, finale: false, own: false, endedAt: ENDED,
            podium: [
                {country: "fr", rank: 1, points: 25}, {country: "pw", rank: 2, points: 18},
                {country: "br", rank: 2, points: 18}, {country: "dz", rank: 3, points: 15},
            ],
            table: [{country: "fr", rank: 1, points: 86}],
        })
    })

    it("keeps a battle apart from a day, its finale too", () => {
        const battle = new Debrief({moment: {case: "battle", value: new BattleEnded({season: 0, number: 2, times: 3, endsSeason: true, roundId: "b"})}})

        expect(dayOf(battle, round("b"))).toMatchObject({season: 0, number: 2, finale: true, own: true})
    })

    it("tells a season's end as its finale's round, numbered as its battle", () => {
        const finale = new BattleEnded({season: 0, number: 3, times: 3, endsSeason: true, roundId: "b"})
        const season = new Debrief({moment: {case: "season", value: new SeasonEnded({
            season: 0, bookId: "season-0", finaleRoundId: "b", finaleBattle: finale,
        })}})
        const battle = new Debrief({moment: {case: "battle", value: finale}})

        expect(dayOf(season, round("b"))).toMatchObject({season: 0, number: 3, finale: true, own: true, endedAt: ENDED})
        expect(dayOf(season, round("b"))).toEqual(dayOf(battle, round("b")))
        expect(dayOf(new Debrief({moment: {case: "season", value: new SeasonEnded({season: 0})}}), round(""))).toBeUndefined()
    })

    it("is none with no debrief, no round, or a round other than the one debriefed", () => {
        expect(dayOf(undefined, round("season-1/day-5"))).toBeUndefined()
        expect(dayOf(new Debrief(), round("season-1/day-5"))).toBeUndefined()
        expect(dayOf(dayFive, undefined)).toBeUndefined()
        expect(dayOf(dayFive, round("season-1/day-4"))).toBeUndefined()
    })
})

describe("the season's finale", () => {
    it("starts with the battle to come that runs to the season's end", () => {
        const finale = Date.parse("2026-10-31T21:00:00Z")
        const timeline = new Timeline({battles: [
            {entryId: "a", startsAt: Timestamp.fromDate(new Date("2026-10-29T18:00:00Z")), times: 2},
            {entryId: "b", startsAt: Timestamp.fromDate(new Date(finale)), times: 3, endsSeason: true},
        ]})

        expect(finaleAtOf(timeline)).toBe(finale)
    })

    it("is none when no battle to come ends the season", () => {
        expect(finaleAtOf(new Timeline({battles: [{entryId: "a", times: 2}]}))).toBeUndefined()
        expect(finaleAtOf(undefined)).toBeUndefined()
    })
})
