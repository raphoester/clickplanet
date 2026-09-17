import {afterEach, describe, expect, it, vi} from "vitest"
import {Code, ConnectError} from "@connectrpc/connect"
import {
    Player as PlayerPb,
    PlayerEvent as PlayerEventPb,
    PlayerLeft as PlayerLeftPb,
    Profile as ProfilePb,
    Roster as RosterPb,
    RosterEntry as RosterEntryPb,
    Stats as StatsPb,
} from "../gen/grpc/player/v1/player_pb.ts"
import {isValidUsername, PlayerError, RosterEvent} from "./player.ts"
import {ConnectPlayerBackend} from "./playerBackend.ts"
import {SESSION_HEADER, SessionProvider, SessionUnavailableError} from "./session.ts"

const refusing = (code: Code) => vi.fn(async () => {
    throw new ConnectError("no", code)
})

const answering = (name: string) => vi.fn(async () => ({profile: new ProfilePb({accountId: "account-1", name})}))

function sessionOf(...tokens: string[]): SessionProvider & {invalidate: ReturnType<typeof vi.fn>} {
    let minted = 0
    return {
        token: vi.fn(async () => tokens[Math.min(minted, tokens.length - 1)]),
        held: vi.fn(() => undefined),
        invalidate: vi.fn(() => {
            minted++
        }),
    }
}

function backendWith(methods: Record<string, unknown>, session: SessionProvider = sessionOf("token-1")) {
    return new ConnectPlayerBackend(methods as never, session, methods as never)
}

const headersOf = (call: ReturnType<typeof vi.fn>, n = 0) =>
    (call.mock.calls[n] as unknown[])[1] as {headers: Headers}

afterEach(() => vi.restoreAllMocks())

describe("isValidUsername", () => {
    it("takes 3 to 20 letters, digits and underscores", () => {
        expect(isValidUsername("ana")).toBe(true)
        expect(isValidUsername("Ana_2024")).toBe(true)
        expect(isValidUsername("x".repeat(20))).toBe(true)
    })

    it("refuses a name too short or too long", () => {
        expect(isValidUsername("ab")).toBe(false)
        expect(isValidUsername("x".repeat(21))).toBe(false)
    })

    it("refuses anything but ASCII letters, digits and underscores", () => {
        expect(isValidUsername("ana bo")).toBe(false)
        expect(isValidUsername("anaïs")).toBe(false)
        expect(isValidUsername("ana-bo")).toBe(false)
    })

    // The chat puts that prefix before every guest's name.
    it("refuses a name that starts like a guest's, in any case", () => {
        expect(isValidUsername("guest_ana")).toBe(false)
        expect(isValidUsername("GUEST_ana")).toBe(false)
        expect(isValidUsername("guestana")).toBe(true)
    })
})

describe("ConnectPlayerBackend", () => {
    it("reads the profile with the click token", async () => {
        const getProfile = answering("ana")
        const backend = backendWith({getProfile})

        expect(await backend.profile()).toEqual({accountId: "account-1", name: "ana"})
        expect(headersOf(getProfile).headers.get(SESSION_HEADER)).toBe("token-1")
    })

    it("reads a profile with no name as an empty name", async () => {
        const backend = backendWith({getProfile: vi.fn(async () => ({}))})

        expect(await backend.profile()).toEqual({accountId: "", name: ""})
    })

    it("sends the name with the click token, and answers what the server stored", async () => {
        const setName = answering("Ana")
        const backend = backendWith({setName})

        expect(await backend.setName("Ana")).toEqual({accountId: "account-1", name: "Ana"})
        expect(setName).toHaveBeenCalledWith({name: "Ana"}, expect.anything())
        expect(headersOf(setName).headers.get(SESSION_HEADER)).toBe("token-1")
    })

    // The token may name the account the browser was on before a sign-in.
    it("sends a refused call once more with a fresh session", async () => {
        const session = sessionOf("stale", "fresh")
        const setName = vi.fn()
            .mockRejectedValueOnce(new ConnectError("no", Code.Unauthenticated))
            .mockResolvedValue({profile: new ProfilePb({name: "ana"})})
        const backend = backendWith({setName}, session)

        expect(await backend.setName("ana")).toMatchObject({name: "ana"})
        expect(session.invalidate).toHaveBeenCalledTimes(1)
        expect(headersOf(setName, 0).headers.get(SESSION_HEADER)).toBe("stale")
        expect(headersOf(setName, 1).headers.get(SESSION_HEADER)).toBe("fresh")
    })

    it("reports a second refusal as not signed in, and does not loop", async () => {
        const getProfile = refusing(Code.Unauthenticated)
        const backend = backendWith({getProfile})

        await expect(backend.profile()).rejects.toMatchObject({failure: "notSignedIn"})
        expect(getProfile).toHaveBeenCalledTimes(2)
    })

    it("maps each refusal to its failure", async () => {
        const cases: [Code, string][] = [
            [Code.InvalidArgument, "invalid"],
            [Code.AlreadyExists, "taken"],
            [Code.PermissionDenied, "guest"],
            [Code.Internal, "failed"],
        ]
        for (const [code, failure] of cases) {
            const backend = backendWith({setName: refusing(code)})

            const error = await backend.setName("ana").catch((e) => e)

            expect(error).toBeInstanceOf(PlayerError)
            expect(error.failure).toBe(failure)
        }
    })

    it("reports a session that could not be minted as failed", async () => {
        const session: SessionProvider = {
            token: vi.fn(async () => {
                throw new SessionUnavailableError()
            }),
            held: vi.fn(() => undefined),
            invalidate: vi.fn(),
        }
        const setName = answering("ana")
        const backend = backendWith({setName}, session)

        await expect(backend.setName("ana")).rejects.toMatchObject({failure: "failed"})
        expect(setName).not.toHaveBeenCalled()
    })

    it("retries the read while the server cannot be reached", async () => {
        vi.spyOn(console, "error").mockImplementation(() => {})
        const getProfile = vi.fn()
            .mockRejectedValueOnce(new ConnectError("down", Code.Unavailable))
            .mockResolvedValue({profile: new ProfilePb({name: "ana"})})
        const backend = backendWith({getProfile})

        expect(await backend.profile()).toMatchObject({name: "ana"})
        expect(getProfile).toHaveBeenCalledTimes(2)
    })

    it("sends a write once, even when the server cannot be reached", async () => {
        const setName = refusing(Code.Unavailable)
        const backend = backendWith({setName})

        await expect(backend.setName("ana")).rejects.toMatchObject({failure: "failed"})
        expect(setName).toHaveBeenCalledTimes(1)
    })
})

describe("ConnectPlayerBackend presence", () => {
    /** A session holding `held`, whose `token` would mint: a test fails if it is called. */
    function holding(held: string | undefined) {
        let current = held
        return {
            token: vi.fn(async () => "minted"),
            held: vi.fn(() => current),
            invalidate: vi.fn(() => {
                current = undefined
            }),
        }
    }

    const presence = {countryCode: "fr", guestName: "Bo"}

    it("announces with the token it holds", async () => {
        const session = holding("token-1")
        const announce = vi.fn(async () => ({}))
        const backend = backendWith({announce}, session)

        expect(await backend.announce(presence)).toBe(true)

        expect(announce).toHaveBeenCalledWith({countryId: "fr", guestName: "Bo"}, expect.anything())
        expect(headersOf(announce).headers.get(SESSION_HEADER)).toBe("token-1")
        expect(session.token).not.toHaveBeenCalled()
    })

    // A mint is a Turnstile check: a visitor who never clicked is not listed.
    it("sends nothing, and mints nothing, while no token is held", async () => {
        const session = holding(undefined)
        const announce = vi.fn(async () => ({}))
        const backend = backendWith({announce}, session)

        expect(await backend.announce(presence)).toBe(false)

        expect(announce).not.toHaveBeenCalled()
        expect(session.token).not.toHaveBeenCalled()
    })

    it("drops a refused token rather than minting a fresh one to retry", async () => {
        const session = holding("stale")
        const announce = refusing(Code.Unauthenticated)
        const backend = backendWith({announce}, session)

        await expect(backend.announce(presence)).rejects.toMatchObject({failure: "notSignedIn"})

        expect(announce).toHaveBeenCalledTimes(1)
        expect(session.invalidate).toHaveBeenCalledTimes(1)
        expect(session.token).not.toHaveBeenCalled()
        expect(backend.heldSession()).toBeUndefined()
    })

    it("reports an unknown country as invalid", async () => {
        const backend = backendWith({announce: refusing(Code.InvalidArgument)}, holding("token-1"))

        await expect(backend.announce(presence)).rejects.toMatchObject({failure: "invalid"})
    })

    it("sends an announce once, even when the server cannot be reached", async () => {
        const announce = refusing(Code.Unavailable)
        const backend = backendWith({announce}, holding("token-1"))

        await expect(backend.announce(presence)).rejects.toBeInstanceOf(PlayerError)
        expect(announce).toHaveBeenCalledTimes(1)
    })

    it("leaves with the token it holds, and sends nothing without one", async () => {
        const leave = vi.fn(async () => ({}))

        backendWith({leave}, holding("token-1")).leave()
        backendWith({leave}, holding(undefined)).leave()

        expect(leave).toHaveBeenCalledTimes(1)
        expect(headersOf(leave).headers.get(SESSION_HEADER)).toBe("token-1")
    })
})

/** A stream that fails before its first event. */
const failingWith = (error: ConnectError) => (): AsyncIterable<PlayerEventPb> => ({
    [Symbol.asyncIterator]: () => ({
        next: async () => {
            throw error
        },
    }),
})

describe("ConnectPlayerBackend live roster", () => {
    const entryPb = new RosterEntryPb({key: "k1", name: "ana", tag: "4f2ca1", countryId: "fr", guest: false, admin: true})
    const ana = {key: "k1", name: "ana", tag: "4f2ca1", countryCode: "fr", guest: false, admin: true}

    afterEach(() => vi.useRealTimers())

    it("follows the stream without a token, maps each event and skips the heartbeats", async () => {
        const listenForEvents = vi.fn(async function* () {
            yield new PlayerEventPb({event: {case: "roster", value: new RosterPb({entries: [entryPb]})}})
            yield new PlayerEventPb({event: {case: "heartbeat", value: {}}})
            yield new PlayerEventPb({event: {case: "entry", value: entryPb}})
            yield new PlayerEventPb({event: {case: "left", value: new PlayerLeftPb({key: "k1"})}})
            await new Promise(() => {})
        })
        const events: RosterEvent[] = []
        const onUnavailable = vi.fn()

        const stop = backendWith({listenForEvents}).listenForRoster((event) => events.push(event), onUnavailable)
        await vi.waitFor(() => expect(events).toHaveLength(3))
        stop()

        expect(events).toEqual([
            {kind: "roster", entries: [ana]},
            {kind: "entry", entry: ana},
            {kind: "left", key: "k1"},
        ])
        expect(onUnavailable).not.toHaveBeenCalled()
        const options = (listenForEvents.mock.calls[0] as unknown[])[1] as {headers?: Headers, timeoutMs: number}
        expect(options.headers).toBeUndefined()
        expect(options.timeoutMs).toBe(0)
    })

    // Connect reads a 404 as unimplemented: a server from before the live roster.
    it("reports a server without the stream once, and does not reconnect to it", async () => {
        vi.useFakeTimers()
        for (const code of [Code.Unimplemented, Code.NotFound]) {
            const listenForEvents = vi.fn(failingWith(new ConnectError("no", code)))
            const onUnavailable = vi.fn()

            backendWith({listenForEvents}).listenForRoster(() => {}, onUnavailable)
            await vi.advanceTimersByTimeAsync(60_000)

            expect(onUnavailable).toHaveBeenCalledTimes(1)
            expect(listenForEvents).toHaveBeenCalledTimes(1)
        }
    })

    it("reconnects after any other failure", async () => {
        vi.useFakeTimers()
        vi.spyOn(console, "error").mockImplementation(() => {})
        const listenForEvents = vi.fn(failingWith(new ConnectError("down", Code.Unavailable)))
        const onUnavailable = vi.fn()

        const stop = backendWith({listenForEvents}).listenForRoster(() => {}, onUnavailable)
        await vi.advanceTimersByTimeAsync(1_000)
        stop()

        expect(listenForEvents.mock.calls.length).toBeGreaterThan(1)
        expect(onUnavailable).not.toHaveBeenCalled()
    })
})

describe("ConnectPlayerBackend player info", () => {
    it("reads a player by name without a token, and maps it", async () => {
        const session = sessionOf("token-1")
        const getPlayer = vi.fn(async () => ({
            player: new PlayerPb({
                name: "Ana",
                stats: new StatsPb({tilesTaken: 1234n, streakCurrent: 3, streakBest: 7, streakLastDay: "2026-09-17"}),
                createdAtUnixMs: 1_788_000_000_000n,
                admin: true,
            }),
        }))

        expect(await backendWith({getPlayer}, session).playerInfo("ana")).toEqual({
            name: "Ana", tilesTaken: 1234, streakCurrent: 3, streakBest: 7, createdAt: 1_788_000_000_000, admin: true,
        })
        expect(getPlayer).toHaveBeenCalledWith({name: "ana"})
        expect(session.token).not.toHaveBeenCalled()
    })

    it("leaves out a creation date the server does not know", async () => {
        const getPlayer = vi.fn(async () => ({player: new PlayerPb({name: "Ana", stats: new StatsPb()})}))

        expect((await backendWith({getPlayer}).playerInfo("Ana"))?.createdAt).toBeUndefined()
    })

    it("answers undefined for a name no player holds", async () => {
        expect(await backendWith({getPlayer: refusing(Code.NotFound)}).playerInfo("Bob")).toBeUndefined()
    })

    it("reports any other failure as it is", async () => {
        const error = await backendWith({getPlayer: refusing(Code.Internal)}).playerInfo("Ana").catch((e) => e)

        expect(error).toBeInstanceOf(ConnectError)
    })
})
