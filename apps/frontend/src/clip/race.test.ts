import {describe, expect, it} from "vitest"
import {protoInt64} from "@bufbuild/protobuf"
import {ClosedRound} from "../gen/grpc/scoring/v1/scoring_pb.ts"
import {dayOf} from "./race.ts"

describe("the day a race closed last", () => {
    it("puts the places that scored on its podium, ties sharing one, and keeps the season after it", () => {
        const closed = new ClosedRound({
            season: 1, number: 5, finale: false, endedAtUnixMs: protoInt64.parse(1_760_734_800_000),
            standings: [
                {rank: 3, countryId: "dz", share: 0.06, points: 15},
                {rank: 1, countryId: "fr", share: 0.12, points: 25},
                {rank: 2, countryId: "pw", share: 0.09, points: 18},
                {rank: 2, countryId: "br", share: 0.09, points: 18},
                {rank: 4, countryId: "ma", share: 0.05, points: 12},
            ],
            after: [{rank: 1, countryId: "fr", points: 86, roundsWon: 3}],
        })

        expect(dayOf(closed)).toEqual({
            season: 1, number: 5, finale: false, own: false, endedAt: 1_760_734_800_000,
            podium: [
                {country: "fr", rank: 1, points: 25}, {country: "pw", rank: 2, points: 18},
                {country: "br", rank: 2, points: 18}, {country: "dz", rank: 3, points: 15},
            ],
            table: [{country: "fr", rank: 1, points: 86}],
        })
    })

    it("keeps a battle's round of its own apart from a day", () => {
        const closed = new ClosedRound({season: 0, number: 1, finale: true, own: true, times: 3, endedAtUnixMs: protoInt64.parse(1_793_487_600_000)})

        expect(dayOf(closed)).toMatchObject({season: 0, number: 1, finale: true, own: true})
    })

    it("is none when no round closed yet", () => {
        expect(dayOf(undefined)).toBeUndefined()
    })
})
