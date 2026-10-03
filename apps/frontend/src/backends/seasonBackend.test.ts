import {describe, expect, it, vi} from "vitest"
import {Code, ConnectError} from "@connectrpc/connect"
import {GetSeasonResponse, Season as SeasonPb} from "../gen/grpc/seasons/v1/seasons_pb.ts"
import {ConnectSeasonBackend} from "./seasonBackend.ts"

const backendWith = (getSeason: () => Promise<GetSeasonResponse>) =>
    new ConnectSeasonBackend({getSeason: vi.fn(getSeason)} as never, "https://api.clickplanet.lol")

describe("ConnectSeasonBackend", () => {
    it("reads the season in milliseconds, with where its finale's calendar file is", async () => {
        const backend = backendWith(async () => new GetSeasonResponse({
            season: new SeasonPb({number: 2, finaleStartsAtUnixMs: 1793480400000n, endsAtUnixMs: 1793487600000n}),
        }))

        expect(await backend.season()).toEqual({
            number: 2,
            finaleStartsAt: 1793480400000,
            endsAt: 1793487600000,
            finaleFile: "https://api.clickplanet.lol/seasons/2/finale.ics",
        })
    })

    it("has no season when the server answers none", async () => {
        expect(await backendWith(async () => new GetSeasonResponse()).season()).toBeUndefined()
    })

    it("has no season on a server without seasons", async () => {
        const backend = backendWith(async () => {
            throw new ConnectError("not found", Code.Unimplemented)
        })

        expect(await backend.season()).toBeUndefined()
    })

    it("asks once per page load", async () => {
        const getSeason = vi.fn(async () => new GetSeasonResponse())
        const backend = new ConnectSeasonBackend({getSeason} as never, "https://api.clickplanet.lol")

        await Promise.all([backend.season(), backend.season()])
        await backend.season()

        expect(getSeason).toHaveBeenCalledTimes(1)
    })

    it("passes on a refusal it does not know", async () => {
        const backend = backendWith(async () => {
            throw new ConnectError("no", Code.PermissionDenied)
        })

        await expect(backend.season()).rejects.toThrow(ConnectError)
    })
})
