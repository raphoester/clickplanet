import {describe, expect, it} from "vitest"
import {Cast} from "../gen/grpc/clips/v1/clips_pb.ts"
import {castOf} from "./cast.ts"

describe("the run's cast", () => {
    const written = Cast.fromJson({
        id: "cast-1", since: "2026-10-28T08:00:00Z", until: "2026-10-28T20:00:00Z",
        fighters: [{number: 1, name: "BraveFox42"}, {number: 2, name: "مقاتل_7"}],
        takes: [
            {at: "2026-10-28T10:00:00.250Z", tileId: 412, countryId: "dz", previousCountryId: "fr", fighter: 2},
            {at: "2026-10-28T10:00:01Z", tileId: 413, countryId: "fr", previousCountryId: "dz", fighter: 1},
        ],
    })

    it("names each player by its number and keeps its takes, for the flag each took for", () => {
        expect(castOf(written)).toEqual({
            id: "cast-1",
            names: new Map([[1, "BraveFox42"], [2, "مقاتل_7"]]),
            takes: [
                {at: Date.parse("2026-10-28T10:00:00.250Z"), tile: 412, flag: "dz", fighter: 2},
                {at: Date.parse("2026-10-28T10:00:01Z"), tile: 413, flag: "fr", fighter: 1},
            ],
        })
    })

    it("is refused with a take of a player it does not name, or with no time", () => {
        expect(() => castOf(new Cast({...written, fighters: written.fighters.slice(0, 1)}))).toThrow("fighter 2, whom it does not name")
        expect(() => castOf(new Cast({...written, takes: [{tileId: 9, countryId: "fr", fighter: 1}]}))).toThrow("tile 9 with no time")
    })
})
