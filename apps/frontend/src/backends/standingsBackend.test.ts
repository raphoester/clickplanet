import {afterEach, describe, expect, it, vi} from "vitest"
import {Code, ConnectError} from "@connectrpc/connect"
import {
    Board,
    GetMySeasonResponse,
    Heartbeat,
    SeasonEvent,
    Standing as StandingPb,
} from "../gen/grpc/seasons/v1/seasons_pb.ts"
import {Rank as RankPb, Title as TitlePb} from "../gen/grpc/player/v1/title_pb.ts"
import {NameColor} from "./player.ts"
import {SESSION_HEADER, SessionProvider} from "./session.ts"
import {Standing} from "./standings.ts"
import {ConnectStandingsBackend} from "./standingsBackend.ts"
import {NO_TIMEOUT} from "./transport.ts"

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

describe("ConnectStandingsBackend.listenForStandings", () => {
    const board = (...standings: StandingPb[]) => new SeasonEvent({event: {case: "board", value: new Board({standings})}})
    const heartbeat = new SeasonEvent({event: {case: "heartbeat", value: new Heartbeat()}})

    const streaming = (...events: SeasonEvent[]) =>
        vi.fn<(req: object, options: {signal: AbortSignal, timeoutMs: number, headers?: Headers}) => AsyncIterable<SeasonEvent>>(
            () => (async function* () {
                yield* events
                await new Promise(() => {})
            })(),
        )

    it("reads each board the stream sends, in numbers, and skips the heartbeats", async () => {
        const listenForEvents = streaming(
            board(
                new StandingPb({rank: 1, name: "Ana", color: NameColor.PINK, countryId: "fr", tiles: 1_204n}),
                new StandingPb({rank: 2, name: "kiran_07", countryId: "in", tiles: 12n}),
            ),
            heartbeat,
            board(new StandingPb({rank: 1, name: "Mateus", color: NameColor.TEAL, countryId: "br", tiles: 1_300n})),
        )
        const seen: unknown[] = []

        const stop = backendWith({listenForEvents}).listenForStandings("", (standings) => seen.push(standings))

        await vi.waitFor(() => expect(seen).toHaveLength(2))
        expect(seen).toEqual([
            [
                {rank: 1, name: "Ana", color: NameColor.PINK, countryCode: "fr", tiles: 1_204},
                {rank: 2, name: "kiran_07", color: NameColor.UNSPECIFIED, countryCode: "in", tiles: 12},
            ],
            [{rank: 1, name: "Mateus", color: NameColor.TEAL, countryCode: "br", tiles: 1_300}],
        ])
        stop()
    })

    it("reads the title each player wears, and none where it wears none", async () => {
        const listenForEvents = streaming(board(
            new StandingPb({rank: 1, name: "Ana", countryId: "fr", tiles: 9n, wornTitle: warlordPb}),
            new StandingPb({rank: 2, name: "kiran_07", countryId: "in", tiles: 3n}),
        ))
        const seen: Standing[][] = []

        const stop = backendWith({listenForEvents}).listenForStandings("", (standings) => seen.push(standings))

        await vi.waitFor(() => expect(seen).toHaveLength(1))
        const [ana, kiran] = seen[0]
        expect(ana.wornTitle).toEqual(warlord)
        expect(kiran.wornTitle).toBeUndefined()
        stop()
    })

    it("follows the players of one country, with no timeout and without a token", async () => {
        const listenForEvents = streaming()
        const session = holding("token-1")

        const stop = backendWith({listenForEvents}, session).listenForStandings("fr", () => {})

        await vi.waitFor(() => expect(listenForEvents).toHaveBeenCalled())
        const [req, options] = listenForEvents.mock.calls[0]
        expect(req).toEqual({countryId: "fr"})
        expect(options.timeoutMs).toBe(NO_TIMEOUT)
        expect(options.headers?.get(SESSION_HEADER)).toBeUndefined()
        expect(session.held).not.toHaveBeenCalled()
        expect(session.token).not.toHaveBeenCalled()
        stop()
    })

    it("closes the stream when told to stop", async () => {
        const listenForEvents = streaming()

        const stop = backendWith({listenForEvents}).listenForStandings("", () => {})
        await vi.waitFor(() => expect(listenForEvents).toHaveBeenCalled())
        stop()

        expect(listenForEvents.mock.calls[0][1].signal.aborted).toBe(true)
    })
})

describe("ConnectStandingsBackend.mySeason", () => {
    it("reads the caller's main flag, its tiles and the rank among all players, in numbers, as its identity", async () => {
        const getMySeason = vi.fn(async () => new GetMySeasonResponse({countryId: "fr", tiles: 340n, globalRank: 12}))

        expect(await backendWith({getMySeason}).mySeason("")).toEqual({countryCode: "fr", tiles: 340, rank: 12})
        expect(getMySeason).toHaveBeenCalledWith({countryId: ""}, expect.anything())
        expect(headersOf(getMySeason).headers.get(SESSION_HEADER)).toBe("token-1")
    })

    it("reads the tiles taken for the country asked and the rank there, whatever the main flag", async () => {
        const getMySeason = vi.fn(async () => new GetMySeasonResponse({
            countryId: "bg", tiles: 74n, globalRank: 1, countryTiles: 3n, countryRank: 2,
        }))

        expect(await backendWith({getMySeason}).mySeason("fr")).toEqual({countryCode: "fr", tiles: 3, rank: 2})
        expect(getMySeason).toHaveBeenCalledWith({countryId: "fr"}, expect.anything())
    })

    it("reads the title the caller wears, on any board", async () => {
        const getMySeason = vi.fn(async () => new GetMySeasonResponse({countryId: "fr", tiles: 340n, globalRank: 12, countryTiles: 3n, countryRank: 3, wornTitle: warlordPb}))

        expect((await backendWith({getMySeason}).mySeason(""))?.wornTitle).toEqual(warlord)
        expect((await backendWith({getMySeason}).mySeason("fr"))?.wornTitle).toEqual(warlord)
    })

    it("reads no flag and no rank where the server answers none", async () => {
        const getMySeason = vi.fn(async () => new GetMySeasonResponse())

        expect(await backendWith({getMySeason}).mySeason("")).toEqual({countryCode: undefined, tiles: 0, rank: undefined})
    })

    it("reads no tiles and no rank in a country the caller took nothing for", async () => {
        const getMySeason = vi.fn(async () => new GetMySeasonResponse({countryId: "fr", tiles: 5n, globalRank: 2}))

        expect(await backendWith({getMySeason}).mySeason("de")).toEqual({countryCode: "de", tiles: 0, rank: undefined})
    })

    it("reads tiles without a rank for a guest", async () => {
        const getMySeason = vi.fn(async () => new GetMySeasonResponse({countryId: "de", tiles: 7n, countryTiles: 7n}))

        expect(await backendWith({getMySeason}).mySeason("")).toEqual({countryCode: "de", tiles: 7, rank: undefined})
        expect(await backendWith({getMySeason}).mySeason("de")).toEqual({countryCode: "de", tiles: 7, rank: undefined})
    })

    it("asks nothing and never mints with no identity to be had", async () => {
        const getMySeason = vi.fn(async () => new GetMySeasonResponse())
        const session = holding(undefined)

        expect(await backendWith({getMySeason}, session).mySeason("")).toBeUndefined()
        expect(getMySeason).not.toHaveBeenCalled()
        expect(session.token).not.toHaveBeenCalled()
    })

    it("knows nothing when the server refuses the token, and keeps it", async () => {
        const session = holding("token-1")

        expect(await backendWith({getMySeason: refusing(Code.Unauthenticated)}, session).mySeason("")).toBeUndefined()
        expect(session.invalidate).not.toHaveBeenCalled()
        expect(session.token).not.toHaveBeenCalled()
    })

    it("knows nothing on a server without standings", async () => {
        expect(await backendWith({getMySeason: refusing(Code.Unimplemented)}).mySeason("")).toBeUndefined()
    })

    it("passes on a refusal it does not know", async () => {
        await expect(backendWith({getMySeason: refusing(Code.Internal)}).mySeason("")).rejects.toThrow(ConnectError)
    })
})
