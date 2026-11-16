import {afterEach, describe, expect, it, vi} from "vitest"
import {Code, ConnectError} from "@connectrpc/connect"
import {BoxEvent, BoxOffered, ClaimBoxResponse, FrozenRefusal, Heartbeat} from "../gen/grpc/boxes/v1/boxes_pb.ts"
import {BonusReward} from "../domain/bonus.ts"
import {BonusLostError, BoxOffer, GrantSource, SessionSource} from "./backend.ts"
import {BOX_REWARD_WAIT_MS, boxOf, BoxesBackend} from "./boxesBackend.ts"
import {NoSession, SESSION_HEADER, type SessionProvider} from "./session.ts"

const heartbeat = () => new BoxEvent({event: {case: "heartbeat", value: new Heartbeat()}})

type BoxesListen = (request: object, options: {signal: AbortSignal, headers: Headers}) => AsyncIterable<BoxEvent>

// A box stream that opens and says nothing, until it is aborted.
function quietBoxes() {
    return vi.fn<BoxesListen>(() => (async function* () {
        await new Promise(() => {})
        yield* [] as BoxEvent[]
    })())
}

function boxesWith(fields: Record<string, unknown> = {}) {
    return {listenForEvents: quietBoxes(), ...fields} as never
}

class Grants implements GrantSource {
    private listeners: ((rewards: BonusReward[]) => void)[] = []

    listenForGrants(onGranted: (rewards: BonusReward[]) => void): () => void {
        this.listeners.push(onGranted)
        return () => {
            this.listeners = this.listeners.filter((listener) => listener !== onGranted)
        }
    }

    grant(...rewards: BonusReward[]) {
        this.listeners.forEach((listener) => listener(rewards))
    }

    get listening() {
        return this.listeners.length
    }
}

class Sessions implements SessionSource {
    private listeners: ((token: string) => void)[] = []

    listenForSessions(onSession: (token: string) => void): () => void {
        this.listeners.push(onSession)
        return () => {
            this.listeners = this.listeners.filter((listener) => listener !== onSession)
        }
    }

    carried(token: string) {
        this.listeners.forEach((listener) => listener(token))
    }
}

function fixedSession(token: string): SessionProvider {
    return {
        token: async () => token, held: () => token, identity: async () => token, heldIdentity: () => token,
        invalidate: () => {},
    }
}

// Each token in turn: a refused one is invalidated, and the next call mints the next.
function rotatingSession(tokens: string[]) {
    let at = 0
    const session = {
        invalidated: 0,
        token: async () => tokens[at],
        held: () => tokens[at], identity: async () => tokens[at], heldIdentity: () => tokens[at],
        invalidate: () => {
            session.invalidated++
            at++
        },
    }
    return session
}

const tokensOpenedWith = (listen: ReturnType<typeof quietBoxes>) =>
    listen.mock.calls.map(call => call[1].headers.get(SESSION_HEADER))

afterEach(() => {
    vi.useRealTimers()
})

describe("boxOf", () => {
    const offered = (fields: {token?: string, seed?: number, expiresAtUnixMs?: bigint} = {}) =>
        new BoxEvent({
            event: {
                case: "boxOffered",
                value: new BoxOffered({token: "a-token", seed: 7, expiresAtUnixMs: 1_000_000n, ...fields}),
            },
        })

    it("reads the box the server addressed to this client, a question mark that says nothing of what it holds", () => {
        expect(boxOf(offered())).toEqual({token: "a-token", seed: 7, expiresAt: expect.any(Number)})
    })

    it("builds the expiry from how long is left, not from the server's clock", () => {
        expect(boxOf(offered({expiresAtUnixMs: 1_015_000n}), 5_000, 1_000_000)?.expiresAt).toBe(5_000 + 15_000)

        const skewed = boxOf(offered({expiresAtUnixMs: 1_015_000n}), 5_000, 1_000_000 + 600_000)
        const honest = boxOf(offered({expiresAtUnixMs: 1_015_000n}), 5_000, 1_000_000)
        expect(skewed?.expiresAt).not.toBe(honest?.expiresAt)
    })

    it("drops everything that is not a box", () => {
        expect(boxOf(heartbeat())).toBeUndefined()
        expect(boxOf(new BoxEvent())).toBeUndefined()
    })
})

describe("BoxesBackend", () => {
    it("hands each box offered on its stream to the box listeners, and nothing else", async () => {
        const listenForEvents = vi.fn(async function* () {
            yield heartbeat()
            yield new BoxEvent({event: {case: "boxOffered", value: new BoxOffered({token: "box-1", expiresAtUnixMs: BigInt(Date.now() + 15_000)})}})
            yield new BoxEvent()
            await new Promise(() => {})
        })
        const backend = new BoxesBackend(boxesWith({listenForEvents}), new Grants(), new Sessions(), new NoSession())
        const seen: BoxOffer[] = []
        backend.listenForBoxes((offer) => seen.push(offer))

        await vi.waitFor(() => expect(seen.map((offer) => offer.token)).toEqual(["box-1"]))
        backend.close()
    })

    it("opens its stream with the identity in hand, and never mints for it", async () => {
        const listenForEvents = quietBoxes()
        const token = vi.fn(async () => "minted")
        const session: SessionProvider = {
            token, held: () => undefined, identity: async () => "identity-1", heldIdentity: () => "identity-1", invalidate: () => {},
        }
        const backend = new BoxesBackend(boxesWith({listenForEvents}), new Grants(), new Sessions(), session)

        await vi.waitFor(() => expect(tokensOpenedWith(listenForEvents)).toEqual(["identity-1"]))
        expect(token).not.toHaveBeenCalled()
        backend.close()
    })

    it("opens it again as the account a later call carried, as a guest signs in, and only once", async () => {
        const listenForEvents = quietBoxes()
        const sessions = new Sessions()
        let held = "guest"
        const session: SessionProvider = {
            token: async () => held, held: () => held, identity: async () => held, heldIdentity: () => held,
            invalidate: () => {},
        }
        const backend = new BoxesBackend(boxesWith({listenForEvents}), new Grants(), sessions, session)
        await vi.waitFor(() => expect(tokensOpenedWith(listenForEvents)).toEqual(["guest"]))

        sessions.carried("guest")
        held = "signed-in"
        sessions.carried("signed-in")
        sessions.carried("signed-in")

        await vi.waitFor(() => expect(tokensOpenedWith(listenForEvents)).toEqual(["guest", "signed-in"]))
        expect(listenForEvents.mock.calls[0][1].signal.aborted).toBe(true)
        backend.close()
    })

    it("opens it again when it ends, as on a server restart", async () => {
        const listenForEvents = vi.fn()
            .mockImplementationOnce(async function* () {
                yield heartbeat()
            })
            .mockImplementation(async function* () {
                yield new BoxEvent({event: {case: "boxOffered", value: new BoxOffered({token: "box-2"})}})
                await new Promise(() => {})
            })
        const backend = new BoxesBackend(boxesWith({listenForEvents}), new Grants(), new Sessions(), new NoSession())
        const seen: string[] = []
        backend.listenForBoxes((offer) => seen.push(offer.token))

        await vi.waitFor(() => expect(seen).toEqual(["box-2"]), {timeout: 3_000})
        expect(listenForEvents).toHaveBeenCalledTimes(2)
        backend.close()
    })

    it("stops it on close", async () => {
        const listenForEvents = quietBoxes()
        const backend = new BoxesBackend(boxesWith({listenForEvents}), new Grants(), new Sessions(), new NoSession())
        await vi.waitFor(() => expect(listenForEvents).toHaveBeenCalledOnce())

        backend.close()

        expect(listenForEvents.mock.calls[0][1].signal.aborted).toBe(true)
    })

    it("claims a box with a session token, and takes the grant planet tells next as what it held", async () => {
        const claimBox = vi.fn().mockResolvedValue(new ClaimBoxResponse())
        const grants = new Grants()
        const backend = new BoxesBackend(boxesWith({claimBox}), grants, new Sessions(), fixedSession("session-1"))

        const claimed = backend.claimBox("box-1")
        await vi.waitFor(() => expect(claimBox).toHaveBeenCalledOnce())
        grants.grant({kind: "encloseClicks", shapes: 2, maxTiles: 25})

        await expect(claimed).resolves.toEqual({kind: "encloseClicks", shapes: 2, maxTiles: 25})
        expect(claimBox).toHaveBeenCalledWith({token: "box-1"}, expect.anything())
        expect((claimBox.mock.calls[0][1] as {headers: Headers}).headers.get(SESSION_HEADER)).toBe("session-1")
        expect(grants.listening).toBe(0)
        backend.close()
    })

    it("says the box held nothing when no grant comes in time: the hand had room for none", async () => {
        vi.useFakeTimers()
        const claimBox = vi.fn().mockResolvedValue(new ClaimBoxResponse())
        const grants = new Grants()
        const backend = new BoxesBackend(boxesWith({claimBox}), grants, new Sessions(), fixedSession("session-1"))

        const claimed = backend.claimBox("box-1")
        await vi.advanceTimersByTimeAsync(BOX_REWARD_WAIT_MS)
        grants.grant({kind: "refill", refills: 1})

        await expect(claimed).resolves.toBeUndefined()
        expect(grants.listening).toBe(0)
        backend.close()
    })

    it("mints a new session and claims once more when the boxes refuse the token", async () => {
        const claimBox = vi.fn()
            .mockRejectedValueOnce(new ConnectError("expired", Code.Unauthenticated))
            .mockResolvedValueOnce(new ClaimBoxResponse())
        const session = rotatingSession(["stale", "fresh"])
        const grants = new Grants()
        const backend = new BoxesBackend(boxesWith({claimBox}), grants, new Sessions(), session)

        const claimed = backend.claimBox("box-1")
        await vi.waitFor(() => expect(claimBox).toHaveBeenCalledTimes(2))
        grants.grant({kind: "refill", refills: 1})
        await claimed

        expect(session.invalidated).toBe(1)
        expect(claimBox.mock.calls.map(([, options]) => (options as {headers: Headers}).headers.get(SESSION_HEADER)))
            .toEqual(["stale", "fresh"])
        backend.close()
    })

    it("reports a box gone as lost, and a box refused while frozen as a failed claim, and waits for no grant", async () => {
        const claimBox = vi.fn()
            .mockRejectedValueOnce(new ConnectError("gone", Code.NotFound))
            .mockRejectedValueOnce(new ConnectError("frozen", Code.FailedPrecondition, undefined, [new FrozenRefusal()]))
        const grants = new Grants()
        const backend = new BoxesBackend(boxesWith({claimBox}), grants, new Sessions(), fixedSession("session-1"))

        await expect(backend.claimBox("box-1")).rejects.toBeInstanceOf(BonusLostError)
        await expect(backend.claimBox("box-2")).rejects.toMatchObject({code: Code.FailedPrecondition})
        expect(grants.listening).toBe(0)
        backend.close()
    })
})
