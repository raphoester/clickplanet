import {describe, expect, it, vi} from "vitest"
import {Code, ConnectError} from "@connectrpc/connect"
import {EndedSeason, GetHallOfFameResponse, Score, Standing} from "../gen/grpc/scoring/v1/scoring_pb.ts"
import {NameColor} from "../gen/grpc/player/v1/color_pb.ts"
import {Title} from "../gen/grpc/player/v1/title_pb.ts"
import {ConnectHallOfFameBackend} from "./hallOfFameBackend.ts"

const backendWith = (getHallOfFame: () => Promise<GetHallOfFameResponse>) =>
    new ConnectHallOfFameBackend({getHallOfFame: vi.fn(getHallOfFame)} as never)

describe("ConnectHallOfFameBackend", () => {
    it("reads each season that ended, its table and its players", async () => {
        const backend = backendWith(async () => new GetHallOfFameResponse({
            seasons: [new EndedSeason({
                number: 0,
                endedAtUnixMs: 1793487600000n,
                scores: [new Score({rank: 1, countryId: "fr", points: 214, roundsWon: 4})],
                standings: [new Standing({
                    rank: 1, accountId: "account-ada", name: "Ada", color: NameColor.PINK, countryId: "fr", tiles: 9840n,
                    wornTitle: new Title({id: "og", name: "OG"}),
                })],
            })],
        }))

        expect(await backend.hallOfFame()).toEqual([{
            number: 0,
            endedAt: 1793487600000,
            scores: [{rank: 1, countryCode: "fr", points: 214, roundsWon: 4}],
            standings: [{
                rank: 1, accountId: "account-ada", name: "Ada", color: NameColor.PINK, countryCode: "fr", tiles: 9840,
                wornTitle: {id: "og", name: "OG"},
            }],
        }])
    })

    it("has no season before one ends", async () => {
        expect(await backendWith(async () => new GetHallOfFameResponse()).hallOfFame()).toEqual([])
    })

    it("has no season on a server without the Hall of Fame", async () => {
        const backend = backendWith(async () => {
            throw new ConnectError("not found", Code.Unimplemented)
        })

        expect(await backend.hallOfFame()).toEqual([])
    })

    it("passes on a refusal it does not know", async () => {
        const backend = backendWith(async () => {
            throw new ConnectError("no", Code.PermissionDenied)
        })

        await expect(backend.hallOfFame()).rejects.toThrow(ConnectError)
    })
})
