import {afterEach, describe, expect, it, vi} from "vitest"
import {Code, ConnectError} from "@connectrpc/connect"
import {
    Board,
    GetMyTallyResponse,
    GetRoundResponse,
    GetScoresResponse,
    Heartbeat,
    Round as RoundPb,
    RoundStanding as RoundStandingPb,
    RoundTakes as RoundTakesPb,
    Score as ScorePb,
    Scores as ScoresPb,
    ScoringEvent,
    Standing as StandingPb,
} from "../gen/grpc/scoring/v1/scoring_pb.ts"
import {Rank as RankPb, Title as TitlePb} from "../gen/grpc/player/v1/title_pb.ts"
import {NameColor} from "./player.ts"
import {SESSION_HEADER, SessionProvider} from "./session.ts"
import {Round, Score, Standing} from "./standings.ts"
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

const BOOK = "season-0"

afterEach(() => vi.restoreAllMocks())

describe("ConnectStandingsBackend.listenForStandings", () => {
    const board = (...standings: StandingPb[]) => new ScoringEvent({event: {case: "board", value: new Board({standings})}})
    const heartbeat = new ScoringEvent({event: {case: "heartbeat", value: new Heartbeat()}})

    const streaming = (...events: ScoringEvent[]) =>
        vi.fn<(req: object, options: {signal: AbortSignal, timeoutMs: number, headers?: Headers}) => AsyncIterable<ScoringEvent>>(
            () => (async function* () {
                yield* events
                await new Promise(() => {})
            })(),
        )

    it("reads each board the stream sends, in numbers, and skips the heartbeats", async () => {
        const listenForEvents = streaming(
            board(
                new StandingPb({rank: 1, accountId: "account-ana", name: "Ana", color: NameColor.PINK, countryId: "fr", tiles: 1_204n}),
                new StandingPb({rank: 2, accountId: "account-kiran", name: "kiran_07", countryId: "in", tiles: 12n}),
            ),
            heartbeat,
            board(new StandingPb({
                rank: 1, accountId: "account-mateus", name: "Mateus", color: NameColor.TEAL, countryId: "br", tiles: 1_300n,
            })),
        )
        const seen: unknown[] = []

        const stop = backendWith({listenForEvents}).listenForStandings(BOOK, "", (standings) => seen.push(standings))

        await vi.waitFor(() => expect(seen).toHaveLength(2))
        expect(seen).toEqual([
            [
                {rank: 1, accountId: "account-ana", name: "Ana", color: NameColor.PINK, countryCode: "fr", tiles: 1_204},
                {rank: 2, accountId: "account-kiran", name: "kiran_07", color: NameColor.UNSPECIFIED, countryCode: "in", tiles: 12},
            ],
            [{rank: 1, accountId: "account-mateus", name: "Mateus", color: NameColor.TEAL, countryCode: "br", tiles: 1_300}],
        ])
        stop()
    })

    it("reads the title each player wears, and none where it wears none", async () => {
        const listenForEvents = streaming(board(
            new StandingPb({rank: 1, name: "Ana", countryId: "fr", tiles: 9n, wornTitle: warlordPb}),
            new StandingPb({rank: 2, name: "kiran_07", countryId: "in", tiles: 3n}),
        ))
        const seen: Standing[][] = []

        const stop = backendWith({listenForEvents}).listenForStandings(BOOK, "", (standings) => seen.push(standings))

        await vi.waitFor(() => expect(seen).toHaveLength(1))
        const [ana, kiran] = seen[0]
        expect(ana.wornTitle).toEqual(warlord)
        expect(kiran.wornTitle).toBeUndefined()
        stop()
    })

    it("follows the players of one country, with no timeout and without a token", async () => {
        const listenForEvents = streaming()
        const session = holding("token-1")

        const stop = backendWith({listenForEvents}, session).listenForStandings(BOOK, "fr", () => {})

        await vi.waitFor(() => expect(listenForEvents).toHaveBeenCalled())
        const [req, options] = listenForEvents.mock.calls[0]
        expect(req).toEqual({bookId: BOOK, countryId: "fr"})
        expect(options.timeoutMs).toBe(NO_TIMEOUT)
        expect(options.headers?.get(SESSION_HEADER)).toBeUndefined()
        expect(session.held).not.toHaveBeenCalled()
        expect(session.token).not.toHaveBeenCalled()
        stop()
    })

    it("closes the stream when told to stop", async () => {
        const listenForEvents = streaming()

        const stop = backendWith({listenForEvents}).listenForStandings(BOOK, "", () => {})
        await vi.waitFor(() => expect(listenForEvents).toHaveBeenCalled())
        stop()

        expect(listenForEvents.mock.calls[0][1].signal.aborted).toBe(true)
    })
})

describe("ConnectStandingsBackend.mySeason", () => {
    it("reads the caller's main flag, its tiles and the rank among all players, in numbers, as its identity", async () => {
        const getMyTally = vi.fn(async () => new GetMyTallyResponse({countryId: "fr", tiles: 340n, globalRank: 12}))

        expect(await backendWith({getMyTally}).mySeason(BOOK, "")).toEqual({countryCode: "fr", tiles: 340, rank: 12})
        expect(getMyTally).toHaveBeenCalledWith({bookId: BOOK, countryId: ""}, expect.anything())
        expect(headersOf(getMyTally).headers.get(SESSION_HEADER)).toBe("token-1")
    })

    it("reads the tiles taken for the country asked and the rank there, whatever the main flag", async () => {
        const getMyTally = vi.fn(async () => new GetMyTallyResponse({
            countryId: "bg", tiles: 74n, globalRank: 1, countryTiles: 3n, countryRank: 2,
        }))

        expect(await backendWith({getMyTally}).mySeason(BOOK, "fr")).toEqual({countryCode: "fr", tiles: 3, rank: 2})
        expect(getMyTally).toHaveBeenCalledWith({bookId: BOOK, countryId: "fr"}, expect.anything())
    })

    it("reads the title the caller wears, on any board", async () => {
        const getMyTally = vi.fn(async () => new GetMyTallyResponse({countryId: "fr", tiles: 340n, globalRank: 12, countryTiles: 3n, countryRank: 3, wornTitle: warlordPb}))

        expect((await backendWith({getMyTally}).mySeason(BOOK, ""))?.wornTitle).toEqual(warlord)
        expect((await backendWith({getMyTally}).mySeason(BOOK, "fr"))?.wornTitle).toEqual(warlord)
    })

    it("reads no flag and no rank where the server answers none", async () => {
        const getMyTally = vi.fn(async () => new GetMyTallyResponse())

        expect(await backendWith({getMyTally}).mySeason(BOOK, "")).toEqual({countryCode: undefined, tiles: 0, rank: undefined})
    })

    it("reads no tiles and no rank in a country the caller took nothing for", async () => {
        const getMyTally = vi.fn(async () => new GetMyTallyResponse({countryId: "fr", tiles: 5n, globalRank: 2}))

        expect(await backendWith({getMyTally}).mySeason(BOOK, "de")).toEqual({countryCode: "de", tiles: 0, rank: undefined})
    })

    it("reads tiles without a rank for a guest", async () => {
        const getMyTally = vi.fn(async () => new GetMyTallyResponse({countryId: "de", tiles: 7n, countryTiles: 7n}))

        expect(await backendWith({getMyTally}).mySeason(BOOK, "")).toEqual({countryCode: "de", tiles: 7, rank: undefined})
        expect(await backendWith({getMyTally}).mySeason(BOOK, "de")).toEqual({countryCode: "de", tiles: 7, rank: undefined})
    })

    it("asks nothing and never mints with no identity to be had", async () => {
        const getMyTally = vi.fn(async () => new GetMyTallyResponse())
        const session = holding(undefined)

        expect(await backendWith({getMyTally}, session).mySeason(BOOK, "")).toBeUndefined()
        expect(getMyTally).not.toHaveBeenCalled()
        expect(session.token).not.toHaveBeenCalled()
    })

    it("knows nothing when the server refuses the token, and keeps it", async () => {
        const session = holding("token-1")

        expect(await backendWith({getMyTally: refusing(Code.Unauthenticated)}, session).mySeason(BOOK, "")).toBeUndefined()
        expect(session.invalidate).not.toHaveBeenCalled()
        expect(session.token).not.toHaveBeenCalled()
    })

    it("knows nothing on a server without standings", async () => {
        expect(await backendWith({getMyTally: refusing(Code.Unimplemented)}).mySeason(BOOK, "")).toBeUndefined()
    })

    it("passes on a refusal it does not know", async () => {
        await expect(backendWith({getMyTally: refusing(Code.Internal)}).mySeason(BOOK, "")).rejects.toThrow(ConnectError)
    })
})

describe("ConnectStandingsBackend.listenForRace", () => {
    const streaming = (...events: ScoringEvent[]) =>
        vi.fn<(req: object, options: {signal: AbortSignal, timeoutMs: number}) => AsyncIterable<ScoringEvent>>(
            () => (async function* () {
                yield* events
                await new Promise(() => {})
            })(),
        )

    it("follows the book and the rounds asked for, reads the scores and each round in numbers, and skips the boards", async () => {
        const listenForEvents = streaming(
            new ScoringEvent({event: {case: "board", value: new Board()}}),
            new ScoringEvent({event: {case: "heartbeat", value: new Heartbeat()}}),
            new ScoringEvent({event: {case: "scores", value: new ScoresPb({
                bookId: BOOK, scores: [new ScorePb({rank: 1, countryId: "de", points: 43, roundsWon: 2})],
            })}}),
            new ScoringEvent({event: {case: "round", value: new RoundPb({
                roundId: "entry-1",
                bookId: BOOK,
                times: 3,
                startedAtUnixMs: 1_792_184_400_000n,
                standings: [new RoundStandingPb({rank: 1, countryId: "fr", share: 0.25, points: 75})],
            })}}),
        )
        const scores: Score[][] = []
        const rounds: Round[] = []

        const stop = backendWith({listenForEvents}).listenForRace(BOOK, ["season-0/day-21", "entry-1"], {
            onScores: (read) => scores.push(read),
            onRound: (read) => rounds.push(read),
        })

        await vi.waitFor(() => expect(rounds).toHaveLength(1))
        expect(listenForEvents).toHaveBeenCalledWith(
            {bookId: BOOK, countryId: "", roundIds: ["season-0/day-21", "entry-1"]},
            expect.objectContaining({timeoutMs: NO_TIMEOUT}),
        )
        expect(scores).toEqual([[{rank: 1, countryCode: "de", points: 43, roundsWon: 2}]])
        expect(rounds).toEqual([{
            roundId: "entry-1",
            times: 3,
            startedAt: 1_792_184_400_000,
            endedAt: 0,
            standings: [{rank: 1, countryCode: "fr", share: 0.25, points: 75}],
            takes: [],
            players: [],
            before: [],
            after: [],
        }])
        stop()
    })
})

describe("ConnectStandingsBackend.round", () => {
    it("reads a scored round's results, the book before and after it, and a battle's numbers", async () => {
        const getRound = vi.fn(async () => new GetRoundResponse({round: new RoundPb({
            roundId: "entry-31",
            bookId: BOOK,
            times: 3,
            startedAtUnixMs: 1_792_098_000_000n,
            endedAtUnixMs: 1_792_105_200_000n,
            standings: [new RoundStandingPb({rank: 1, countryId: "fr", share: 0.3, points: 75})],
            takes: [new RoundTakesPb({countryId: "fr", tiles: 4210n}), new RoundTakesPb({countryId: "de", tiles: 3105n})],
            players: [new StandingPb({rank: 1, accountId: "account-ana", name: "Ana", countryId: "fr", tiles: 912n})],
            before: [new ScorePb({rank: 1, countryId: "de", points: 43, roundsWon: 2})],
            after: [
                new ScorePb({rank: 1, countryId: "fr", points: 93, roundsWon: 2}),
                new ScorePb({rank: 2, countryId: "de", points: 43, roundsWon: 2}),
            ],
        })}))

        expect(await backendWith({getRound}).round("entry-31")).toEqual({
            roundId: "entry-31",
            times: 3,
            startedAt: 1_792_098_000_000,
            endedAt: 1_792_105_200_000,
            standings: [{rank: 1, countryCode: "fr", share: 0.3, points: 75}],
            takes: [{countryCode: "fr", tiles: 4210}, {countryCode: "de", tiles: 3105}],
            players: [{rank: 1, accountId: "account-ana", name: "Ana", color: NameColor.UNSPECIFIED, countryCode: "fr", tiles: 912, wornTitle: undefined}],
            before: [{rank: 1, countryCode: "de", points: 43, roundsWon: 2}],
            after: [
                {rank: 1, countryCode: "fr", points: 93, roundsWon: 2},
                {rank: 2, countryCode: "de", points: 43, roundsWon: 2},
            ],
        })
        expect(getRound).toHaveBeenCalledWith({roundId: "entry-31"})
    })

    it("reads no round for one never opened", async () => {
        expect(await backendWith({getRound: refusing(Code.NotFound)}).round("entry-9")).toBeUndefined()
    })
})

describe("ConnectStandingsBackend.scores", () => {
    it("reads the book's countries by points", async () => {
        const getScores = vi.fn(async () => new GetScoresResponse({scores: new ScoresPb({
            bookId: BOOK, scores: [new ScorePb({rank: 1, countryId: "fr", points: 93, roundsWon: 2})],
        })}))

        expect(await backendWith({getScores}).scores(BOOK)).toEqual([{rank: 1, countryCode: "fr", points: 93, roundsWon: 2}])
        expect(getScores).toHaveBeenCalledWith({bookId: BOOK})
    })
})
