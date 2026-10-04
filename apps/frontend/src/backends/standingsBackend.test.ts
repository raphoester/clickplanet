import {afterEach, describe, expect, it, vi} from "vitest"
import {Code, ConnectError} from "@connectrpc/connect"
import {
    GetMySeasonResponse,
    GetStandingsResponse,
    Standing as StandingPb,
} from "../gen/grpc/seasons/v1/seasons_pb.ts"
import {Rank as RankPb, Title as TitlePb} from "../gen/grpc/player/v1/title_pb.ts"
import {NameColor} from "./player.ts"
import {SESSION_HEADER, SessionProvider} from "./session.ts"
import {ConnectStandingsBackend} from "./standingsBackend.ts"

const refusing = (code: Code) => vi.fn(async () => {
    throw new ConnectError("no", code)
})

function holding(token: string | undefined): SessionProvider & {token: ReturnType<typeof vi.fn>, held: ReturnType<typeof vi.fn>} {
    return {
        token: vi.fn(async () => "minted"),
        held: vi.fn(() => token),
        identity: vi.fn(async () => token),
        heldIdentity: vi.fn(() => token),
        invalidate: vi.fn(),
    }
}

const backendWith = (methods: Record<string, unknown>, session: SessionProvider = holding("token-1")) =>
    new ConnectStandingsBackend(methods as never, session)

const headersOf = (call: ReturnType<typeof vi.fn>) =>
    (call.mock.calls[0] as unknown[])[1] as {headers: Headers}

const warlordPb = new TitlePb({
    id: "warlord",
    name: "Warlord",
    rank: new RankPb({trackId: "conquest", trackName: "Conquest", number: 3, count: 5}),
})
const warlord = {id: "warlord", name: "Warlord", rank: {trackId: "conquest", trackName: "Conquest", number: 3, count: 5}}

afterEach(() => vi.restoreAllMocks())

describe("ConnectStandingsBackend.standings", () => {
    it("reads each player's rank, name, color, main flag and tiles, in numbers", async () => {
        const getStandings = vi.fn(async () => new GetStandingsResponse({
            standings: [
                new StandingPb({rank: 1, name: "Ana", color: NameColor.PINK, countryId: "fr", tiles: 1_204n}),
                new StandingPb({rank: 2, name: "kiran_07", countryId: "in", tiles: 12n}),
                new StandingPb({rank: 2, name: "Mateus", color: NameColor.TEAL, countryId: "br", tiles: 12n}),
            ],
        }))

        expect(await backendWith({getStandings}).standings("")).toEqual([
            {rank: 1, name: "Ana", color: NameColor.PINK, countryCode: "fr", tiles: 1_204},
            {rank: 2, name: "kiran_07", color: NameColor.UNSPECIFIED, countryCode: "in", tiles: 12},
            {rank: 2, name: "Mateus", color: NameColor.TEAL, countryCode: "br", tiles: 12},
        ])
    })

    it("reads the title each player wears, and none where it wears none", async () => {
        const getStandings = vi.fn(async () => new GetStandingsResponse({
            standings: [
                new StandingPb({rank: 1, name: "Ana", countryId: "fr", tiles: 9n, wornTitle: warlordPb}),
                new StandingPb({rank: 2, name: "kiran_07", countryId: "in", tiles: 3n}),
            ],
        }))

        const [ana, kiran] = await backendWith({getStandings}).standings("")

        expect(ana.wornTitle).toEqual(warlord)
        expect(kiran.wornTitle).toBeUndefined()
    })

    it("asks for the players of one country, without a token", async () => {
        const getStandings = vi.fn(async () => new GetStandingsResponse())
        const session = holding("token-1")

        await backendWith({getStandings}, session).standings("fr")

        expect(getStandings).toHaveBeenCalledWith({countryId: "fr"})
        expect(session.held).not.toHaveBeenCalled()
        expect(session.token).not.toHaveBeenCalled()
    })

    it("has nobody on a server without standings", async () => {
        expect(await backendWith({getStandings: refusing(Code.Unimplemented)}).standings("")).toEqual([])
        expect(await backendWith({getStandings: refusing(Code.NotFound)}).standings("")).toEqual([])
    })

    it("passes on a refusal it does not know", async () => {
        await expect(backendWith({getStandings: refusing(Code.PermissionDenied)}).standings("")).rejects.toThrow(ConnectError)
    })
})

describe("ConnectStandingsBackend.mySeason", () => {
    it("reads the caller's main flag, tiles and ranks, in numbers, as its identity", async () => {
        const getMySeason = vi.fn(async () => new GetMySeasonResponse({countryId: "fr", tiles: 340n, globalRank: 12, countryRank: 3}))

        expect(await backendWith({getMySeason}).mySeason()).toEqual({countryCode: "fr", tiles: 340, globalRank: 12, countryRank: 3})
        expect(headersOf(getMySeason).headers.get(SESSION_HEADER)).toBe("token-1")
    })

    it("reads the title the caller wears", async () => {
        const getMySeason = vi.fn(async () => new GetMySeasonResponse({countryId: "fr", tiles: 340n, globalRank: 12, countryRank: 3, wornTitle: warlordPb}))

        expect((await backendWith({getMySeason}).mySeason())?.wornTitle).toEqual(warlord)
    })

    it("reads no flag and no rank where the server answers none", async () => {
        const getMySeason = vi.fn(async () => new GetMySeasonResponse())

        expect(await backendWith({getMySeason}).mySeason()).toEqual({
            countryCode: undefined,
            tiles: 0,
            globalRank: undefined,
            countryRank: undefined,
        })
    })

    it("reads tiles without a rank for a guest", async () => {
        const getMySeason = vi.fn(async () => new GetMySeasonResponse({countryId: "de", tiles: 7n}))

        expect(await backendWith({getMySeason}).mySeason()).toEqual({
            countryCode: "de",
            tiles: 7,
            globalRank: undefined,
            countryRank: undefined,
        })
    })

    it("asks nothing and never mints with no identity to be had", async () => {
        const getMySeason = vi.fn(async () => new GetMySeasonResponse())
        const session = holding(undefined)

        expect(await backendWith({getMySeason}, session).mySeason()).toBeUndefined()
        expect(getMySeason).not.toHaveBeenCalled()
        expect(session.token).not.toHaveBeenCalled()
    })

    it("knows nothing when the server refuses the token, and keeps it", async () => {
        const session = holding("token-1")

        expect(await backendWith({getMySeason: refusing(Code.Unauthenticated)}, session).mySeason()).toBeUndefined()
        expect(session.invalidate).not.toHaveBeenCalled()
        expect(session.token).not.toHaveBeenCalled()
    })

    it("knows nothing on a server without standings", async () => {
        expect(await backendWith({getMySeason: refusing(Code.Unimplemented)}).mySeason()).toBeUndefined()
    })

    it("passes on a refusal it does not know", async () => {
        await expect(backendWith({getMySeason: refusing(Code.Internal)}).mySeason()).rejects.toThrow(ConnectError)
    })
})
