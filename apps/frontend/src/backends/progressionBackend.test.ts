import {afterEach, describe, expect, it, vi} from "vitest"
import {Code, ConnectError} from "@connectrpc/connect"
import {
    Badge as BadgePb,
    CountryTiles as CountryTilesPb,
    Progress as ProgressPb,
    ProgressionEvent as ProgressionEventPb,
    Stats as StatsPb,
    Step as StepPb,
    TitleEarned as TitleEarnedPb,
    Track as TrackPb,
} from "../gen/grpc/progression/v1/progression_pb.ts"
import {Rank as RankPb, Title as TitlePb} from "../gen/grpc/progression/v1/title_pb.ts"
import {PlayerTitle} from "./progression.ts"
import {ConnectProgressionBackend} from "./progressionBackend.ts"
import {SESSION_HEADER, SessionProvider} from "./session.ts"

const settlerPb = new TitlePb({id: "settler", name: "Settler", rank: new RankPb({trackId: "conquest", trackName: "Conquest", number: 1, count: 5})})
const settler: PlayerTitle = {id: "settler", name: "Settler", rank: {trackId: "conquest", trackName: "Conquest", number: 1, count: 5}}

const refusing = (code: Code) => vi.fn(async () => {
    throw new ConnectError("no", code)
})

function sessionOf(...tokens: string[]): SessionProvider & {invalidate: ReturnType<typeof vi.fn>, token: ReturnType<typeof vi.fn>} {
    let minted = 0
    return {
        token: vi.fn(async () => tokens[Math.min(minted, tokens.length - 1)]),
        held: vi.fn(() => undefined), identity: vi.fn(async () => undefined), heldIdentity: vi.fn(() => undefined),
        invalidate: vi.fn(() => {
            minted++
        }),
    }
}

function holding(identity: string | undefined) {
    return {...sessionOf("minted"), heldIdentity: vi.fn(() => identity)}
}

function backendWith(methods: Record<string, unknown>, session: SessionProvider = sessionOf("token-1")) {
    return new ConnectProgressionBackend(methods as never, session)
}

const headersOf = (call: ReturnType<typeof vi.fn>, n = 0) =>
    (call.mock.calls[n] as unknown[])[1] as {headers: Headers}

afterEach(() => {
    vi.restoreAllMocks()
    vi.useRealTimers()
})

describe("ConnectProgressionBackend", () => {
    it("reads the titles with the click token: the one worn, the ones to wear and each track", async () => {
        const getTitles = vi.fn(async () => ({
            worn: settlerPb,
            wearable: [new TitlePb({id: "og", name: "OG"}), settlerPb],
            tracks: [new TrackPb({
                id: "conquest", name: "Conquest", progress: 140n, steps: [
                    new StepPb({title: settlerPb, threshold: 100n, earned: true}),
                    new StepPb({title: new TitlePb({id: "raider", name: "Raider", rank: new RankPb({trackId: "conquest", trackName: "Conquest", number: 2, count: 5})}), threshold: 1000n}),
                ],
            })],
        }))
        const backend = backendWith({getTitles})

        expect(await backend.titles()).toEqual({
            worn: settler,
            wearable: [{id: "og", name: "OG"}, settler],
            tracks: [{
                id: "conquest", name: "Conquest", progress: 140, steps: [
                    {title: settler, threshold: 100, earned: true},
                    {title: {id: "raider", name: "Raider", rank: {trackId: "conquest", trackName: "Conquest", number: 2, count: 5}}, threshold: 1000, earned: false},
                ],
            }],
        })
        expect(headersOf(getTitles).headers.get(SESSION_HEADER)).toBe("token-1")
    })

    it("reads no worn title as none", async () => {
        const backend = backendWith({getTitles: vi.fn(async () => ({wearable: [], tracks: []}))})

        expect((await backend.titles()).worn).toBeUndefined()
    })

    it("reads every country the player plays for and against with the click token", async () => {
        const getFronts = vi.fn(async () => ({
            playsFor: [new CountryTilesPb({countryId: "fr", tiles: 1200n})],
            playsAgainst: [new CountryTilesPb({countryId: "de", tiles: 900n}), new CountryTilesPb({countryId: "es", tiles: 4n})],
        }))
        const backend = backendWith({getFronts})

        expect(await backend.fronts()).toEqual({
            playsFor: [{countryCode: "fr", tiles: 1200}],
            playsAgainst: [{countryCode: "de", tiles: 900}, {countryCode: "es", tiles: 4}],
        })
        expect(headersOf(getFronts).headers.get(SESSION_HEADER)).toBe("token-1")
    })

    it("wears a title with the click token, and answers the one the server says is worn", async () => {
        const wearTitle = vi.fn(async () => ({worn: settlerPb}))
        const backend = backendWith({wearTitle})

        expect(await backend.wearTitle("settler")).toEqual(settler)
        expect(wearTitle).toHaveBeenCalledWith({titleId: "settler"}, expect.anything())
        expect(headersOf(wearTitle).headers.get(SESSION_HEADER)).toBe("token-1")
    })

    it("sends a refused call once more with a fresh session, and no more", async () => {
        const session = sessionOf("stale", "fresh")
        const wearTitle = vi.fn()
            .mockRejectedValueOnce(new ConnectError("no", Code.Unauthenticated))
            .mockResolvedValue({worn: settlerPb})

        expect(await backendWith({wearTitle}, session).wearTitle("settler")).toEqual(settler)
        expect(session.invalidate).toHaveBeenCalledTimes(1)
        expect(headersOf(wearTitle, 0).headers.get(SESSION_HEADER)).toBe("stale")
        expect(headersOf(wearTitle, 1).headers.get(SESSION_HEADER)).toBe("fresh")

        const refused = refusing(Code.Unauthenticated)
        await expect(backendWith({wearTitle: refused}).wearTitle("settler")).rejects.toBeInstanceOf(ConnectError)
        expect(refused).toHaveBeenCalledTimes(2)
    })

    it("sends a wear once, even when the server cannot be reached", async () => {
        const wearTitle = refusing(Code.Unavailable)

        await expect(backendWith({wearTitle}).wearTitle("settler")).rejects.toBeInstanceOf(ConnectError)
        expect(wearTitle).toHaveBeenCalledTimes(1)
    })
})

describe("ConnectProgressionBackend badges", () => {
    it("reads the badges of the accounts asked in one call, without a token, and maps them", async () => {
        const session = sessionOf("token-1")
        const getBadges = vi.fn(async () => ({
            badges: [new BadgePb({accountId: "account-ana", streak: 12, wornTitle: settlerPb}), new BadgePb({accountId: "account-kiran", streak: 3})],
        }))

        expect(await backendWith({getBadges}, session).badges(["account-ana", "account-kiran", "account-guest"])).toEqual([
            {accountId: "account-ana", streak: 12, wornTitle: settler},
            {accountId: "account-kiran", streak: 3, wornTitle: undefined},
        ])
        expect(getBadges).toHaveBeenCalledWith({accountIds: ["account-ana", "account-kiran", "account-guest"]})
        expect(session.token).not.toHaveBeenCalled()
    })

    it("reports a failure as it is", async () => {
        const error = await backendWith({getBadges: refusing(Code.Unavailable)}).badges(["account-ana"]).catch((e) => e)

        expect(error).toBeInstanceOf(ConnectError)
    })
})

describe("ConnectProgressionBackend public card", () => {
    it("reads a player's progress by its account id without a token, and maps it", async () => {
        const session = sessionOf("token-1")
        const getProgress = vi.fn(async () => ({
            progress: new ProgressPb({
                stats: new StatsPb({tilesTaken: 1234n, streakCurrent: 3, streakBest: 7, streakLastDay: "2026-09-17"}),
                titles: [new TitlePb({id: "og", name: "OG", feat: "Joined before November 2026"}), settlerPb],
                wornTitle: settlerPb,
                playsFor: [new CountryTilesPb({countryId: "fr", tiles: 1200n}), new CountryTilesPb({countryId: "be", tiles: 34n})],
                playsAgainst: [new CountryTilesPb({countryId: "de", tiles: 900n})],
            }),
        }))

        expect(await backendWith({getProgress}, session).progress("account-1")).toEqual({
            tilesTaken: 1234, streakCurrent: 3, streakBest: 7,
            titles: [{id: "og", name: "OG", feat: "Joined before November 2026"}, settler],
            wornTitle: settler,
            playsFor: [{countryCode: "fr", tiles: 1200}, {countryCode: "be", tiles: 34}],
            playsAgainst: [{countryCode: "de", tiles: 900}],
        })
        expect(getProgress).toHaveBeenCalledWith({accountId: "account-1"})
        expect(session.token).not.toHaveBeenCalled()
    })

    it("reads no stats as zeros", async () => {
        const getProgress = vi.fn(async () => ({progress: new ProgressPb()}))

        expect(await backendWith({getProgress}).progress("account-1")).toEqual({
            tilesTaken: 0, streakCurrent: 0, streakBest: 0, titles: [], wornTitle: undefined, playsFor: [], playsAgainst: [],
        })
    })

    it("answers undefined for an account the other players do not see", async () => {
        expect(await backendWith({getProgress: refusing(Code.NotFound)}).progress("account-guest")).toBeUndefined()
    })

    it("retries the read while the server cannot be reached", async () => {
        vi.spyOn(console, "error").mockImplementation(() => {})
        const getProgress = vi.fn()
            .mockRejectedValueOnce(new ConnectError("down", Code.Unavailable))
            .mockResolvedValue({progress: new ProgressPb({stats: new StatsPb({tilesTaken: 5n})})})

        expect(await backendWith({getProgress}).progress("account-1")).toMatchObject({tilesTaken: 5})
        expect(getProgress).toHaveBeenCalledTimes(2)
    })

    it("reports any other failure as it is", async () => {
        const error = await backendWith({getProgress: refusing(Code.Internal)}).progress("account-1").catch((e) => e)

        expect(error).toBeInstanceOf(ConnectError)
    })
})

const failingWith = (error: ConnectError) => (): AsyncIterable<ProgressionEventPb> => ({
    [Symbol.asyncIterator]: () => ({
        next: async () => {
            throw error
        },
    }),
})

describe("ConnectProgressionBackend titles earned", () => {
    it("opens the stream with the token held, never a mint, hands each title earned and skips the heartbeats", async () => {
        const session = holding("identity-1")
        const listenForEvents = vi.fn(async function* () {
            yield new ProgressionEventPb({event: {case: "heartbeat", value: {}}})
            yield new ProgressionEventPb({event: {case: "titleEarned", value: new TitleEarnedPb({title: settlerPb})}})
            await new Promise(() => {})
        })
        const titles: PlayerTitle[] = []

        const stop = backendWith({listenForEvents}, session).listenForTitles((title) => titles.push(title))
        await vi.waitFor(() => expect(titles).toEqual([settler]))
        stop()

        const options = (listenForEvents.mock.calls[0] as unknown[])[1] as {headers: Headers, timeoutMs: number}
        expect(options.headers.get(SESSION_HEADER)).toBe("identity-1")
        expect(options.timeoutMs).toBe(0)
        expect(session.token).not.toHaveBeenCalled()
    })

    it("stops for good on a refused token", async () => {
        vi.useFakeTimers()
        const listenForEvents = vi.fn(failingWith(new ConnectError("no", Code.Unauthenticated)))
        const session = holding("expired")

        backendWith({listenForEvents}, session).listenForTitles(() => {})
        await vi.advanceTimersByTimeAsync(60_000)

        expect(listenForEvents).toHaveBeenCalledTimes(1)
        expect(session.invalidate).not.toHaveBeenCalled()
    })

    it("reconnects after any other failure", async () => {
        vi.useFakeTimers()
        vi.spyOn(console, "error").mockImplementation(() => {})
        const listenForEvents = vi.fn(failingWith(new ConnectError("down", Code.Unavailable)))

        const stop = backendWith({listenForEvents}, holding("identity-1")).listenForTitles(() => {})
        await vi.advanceTimersByTimeAsync(1_000)
        stop()

        expect(listenForEvents.mock.calls.length).toBeGreaterThan(1)
    })
})
