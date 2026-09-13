import {describe, expect, it, vi} from "vitest"
import {asBonusError, bindingsOf, bombOf, catchOf, enclosureOf, offerOf, PlanetBackend, spreadOf, updateOf} from "./planetBackend.ts"
import {Code, ConnectError} from "@connectrpc/connect"
import {
    BombDropped,
    BonusKind,
    BonusOffered,
    BonusTaken,
    ClickBudget as ClickBudgetMessage,
    GetMapResponse,
    GlobePoint,
    Heartbeat,
    PlanetEvent,
    TilesEnclosed,
    TilesSpread,
    TileUpdate,
} from "../gen/grpc/planet/v1/planet_pb.ts"
import {BonusLostError, RateLimitedError, VPNBlockedError} from "./backend.ts"
import {SESSION_HEADER, type SessionProvider, SessionUnavailableError} from "./session.ts"
import type {ClickBudget} from "./clickBudget.ts"

function fixedSession(token: string): SessionProvider {
    return {token: async () => token, invalidate: () => {}}
}

function rotatingSession(tokens: string[]) {
    let index = 0
    return {
        invalidated: 0,
        async token() {
            return tokens[Math.min(index, tokens.length - 1)]
        },
        invalidate() {
            this.invalidated++
            index++
        },
    }
}

function failingSession(): SessionProvider {
    return {
        token: async () => {
            throw new SessionUnavailableError()
        },
        invalidate: () => {},
    }
}

function tileUpdateEvent(fields: {tileId: number, countryId: string, previousCountryId?: string, boosted?: boolean}): PlanetEvent {
    return new PlanetEvent({event: {case: "tileUpdate", value: new TileUpdate(fields)}})
}

describe("updateOf", () => {
    it("maps a tile update onto the shape the globe consumes", () => {
        expect(updateOf(tileUpdateEvent({tileId: 7, countryId: "jp", previousCountryId: "fr"})))
            .toEqual({tile: 7, previousCountry: "fr", newCountry: "jp", boosted: false})
    })

    it("reports an unowned previous tile as undefined rather than an empty code", () => {
        expect(updateOf(tileUpdateEvent({tileId: 1, countryId: "fr"})))
            .toEqual({tile: 1, previousCountry: undefined, newCountry: "fr", boosted: false})
    })

    it("says when the click that made it was boosted", () => {
        expect(updateOf(tileUpdateEvent({tileId: 7, countryId: "jp", boosted: true}))?.boosted).toBe(true)
    })

    it("drops a heartbeat", () => {
        const heartbeat = new PlanetEvent({event: {case: "heartbeat", value: new Heartbeat()}})
        expect(updateOf(heartbeat)).toBeUndefined()
    })

    it("drops an event case this build does not know", () => {
        // What a client sees when the backend adds a case: an unset oneof, not a crash.
        expect(updateOf(new PlanetEvent())).toBeUndefined()
    })
})

function tileBytes(codes: number[]): Uint8Array {
    const bytes = new Uint8Array(codes.length * 2)
    const view = new DataView(bytes.buffer)
    codes.forEach((code, i) => view.setUint16(i * 2, code, true))
    return bytes
}

function mapResponse(startTileId: number, codes: string[], tiles: number[]): GetMapResponse {
    return new GetMapResponse({startTileId, codes, tiles: tileBytes(tiles)})
}

describe("bindingsOf", () => {
    it("resolves tiles against the table that came with them", () => {
        expect(bindingsOf(mapResponse(10, ["", "fr", "de"], [1, 2, 1])))
            .toEqual(new Map([[10, "fr"], [11, "de"], [12, "fr"]]))
    })

    it("skips unowned tiles rather than binding them to an empty code", () => {
        expect(bindingsOf(mapResponse(0, ["", "fr"], [0, 1, 0]))).toEqual(new Map([[1, "fr"]]))
    })

    it("reads codes longer than two characters", () => {
        expect(bindingsOf(mapResponse(0, ["", "gb-eng"], [1]))).toEqual(new Map([[0, "gb-eng"]]))
    })

    it("handles an empty range", () => {
        expect(bindingsOf(mapResponse(0, [""], []))).toEqual(new Map())
    })

    it("survives tiles that do not start at offset zero", () => {
        const padded = new Uint8Array([0xff, 0xff, ...tileBytes([1])])
        const res = new GetMapResponse({startTileId: 4, codes: ["", "jp"], tiles: padded.subarray(2)})
        expect(bindingsOf(res)).toEqual(new Map([[4, "jp"]]))
    })
})

/** A server that throttles nothing, which is what most of these tests are. */
function noBudget() {
    return vi.fn().mockResolvedValue({})
}

/**
 * A live stream that ends at once. The constructor opens one, so a client stub
 * without it makes openStream report a failure these tests are not about.
 */
function noEvents() {
    return vi.fn(async function* () {})
}

type GetMapRequestFields = {startTileId: number, endTileId: number}
type GetMapMock = ReturnType<typeof getMapMock>

function getMapMock(impl?: (req: GetMapRequestFields) => Promise<GetMapResponse>) {
    return vi.fn<(req: GetMapRequestFields) => Promise<GetMapResponse>>(impl)
}

describe("PlanetBackend.getCurrentOwnershipsByBatch", () => {
    const backendWith = (getMap: GetMapMock) => {
        const client = {click: vi.fn(), getMap, getBudget: noBudget(), mapDensity: vi.fn(), listenForEvents: noEvents()} as never
        return new PlanetBackend(client, 1_000)
    }

    const collect = async (backend: PlanetBackend, signal?: AbortSignal) => {
        const seen: Map<number, string>[] = []
        await backend.getCurrentOwnershipsByBatch(2, 4, o => seen.push(o.bindings), signal)
        return seen
    }

    it("walks the range one chunk at a time", async () => {
        const getMap = getMapMock(async () => mapResponse(1, ["", "fr"], [1, 0]))
        const backend = backendWith(getMap)

        expect(await collect(backend)).toEqual([new Map([[1, "fr"]]), new Map([[1, "fr"]])])
        expect(getMap.mock.calls.map(c => c[0])).toEqual([
            {startTileId: 1, endTileId: 3},
            {startTileId: 3, endTileId: 4},
        ])
        backend.close()
    })

    it("retries a server it could not reach", async () => {
        vi.spyOn(console, "error").mockImplementation(() => {})
        const getMap = getMapMock()
            .mockRejectedValueOnce(new ConnectError("offline", Code.Unavailable))
            .mockImplementation(async () => mapResponse(1, [""], [0, 0]))
        const backend = backendWith(getMap)

        await collect(backend)
        expect(getMap).toHaveBeenCalledTimes(3)
        backend.close()
    })

    it("does not retry an error the server answered with", async () => {
        const getMap = getMapMock().mockRejectedValue(new ConnectError("nope", Code.InvalidArgument))
        const backend = backendWith(getMap)

        await expect(collect(backend)).rejects.toThrow(/nope/)
        expect(getMap).toHaveBeenCalledTimes(1)
        backend.close()
    })

    it("stops once the caller aborts", async () => {
        vi.spyOn(console, "error").mockImplementation(() => {})
        const controller = new AbortController()
        const getMap = getMapMock(async () => {
            controller.abort()
            throw new ConnectError("offline", Code.Unavailable)
        })
        const backend = backendWith(getMap)

        await expect(collect(backend, controller.signal)).rejects.toThrow()
        expect(getMap).toHaveBeenCalledTimes(1)
        backend.close()
    })
})

describe("PlanetBackend.clickTile", () => {
    const backendWith = (click: ReturnType<typeof vi.fn>, session?: SessionProvider) => {
        const clientStub = {click, getMap: vi.fn(), getBudget: noBudget(), mapDensity: vi.fn(), listenForEvents: noEvents()} as never
        return new PlanetBackend(clientStub, 1_000, session)
    }

    const headersOf = (click: ReturnType<typeof vi.fn>, call = 0) =>
        (click.mock.calls[call][1] as {headers: Headers}).headers

    it("sends the tile and the country", async () => {
        const click = vi.fn().mockResolvedValue({})
        const backend = backendWith(click)

        await backend.clickTile(42, "fr")
        expect(click).toHaveBeenCalledWith({tileId: 42, countryId: "fr"}, expect.anything())
        backend.close()
    })

    it("sends no session header when this build has no session to send", async () => {
        const click = vi.fn().mockResolvedValue({})
        const backend = backendWith(click)

        await backend.clickTile(42, "fr")

        expect(headersOf(click).has(SESSION_HEADER)).toBe(false)
        backend.close()
    })

    it("puts the session token on the click", async () => {
        const click = vi.fn().mockResolvedValue({})
        const backend = backendWith(click, fixedSession("session-1"))

        await backend.clickTile(42, "fr")

        expect(headersOf(click).get(SESSION_HEADER)).toBe("session-1")
        backend.close()
    })

    // A token that lapsed mid-session, or one bound to an address that changed
    // when a phone moved onto cellular, is not worth a dialog: mint and retry.
    it("mints a new session and retries once when the server refuses the token", async () => {
        const click = vi.fn()
            .mockRejectedValueOnce(new ConnectError("no session", Code.Unauthenticated))
            .mockResolvedValueOnce({})

        const session = rotatingSession(["stale", "fresh"])
        const backend = backendWith(click, session)

        await backend.clickTile(42, "fr")

        expect(click).toHaveBeenCalledTimes(2)
        expect(headersOf(click, 0).get(SESSION_HEADER)).toBe("stale")
        expect(headersOf(click, 1).get(SESSION_HEADER)).toBe("fresh")
        expect(session.invalidated).toBe(1)
        backend.close()
    })

    it("gives up after the retry rather than looping", async () => {
        const click = vi.fn().mockRejectedValue(new ConnectError("no session", Code.Unauthenticated))
        const backend = backendWith(click, rotatingSession(["stale", "fresh"]))

        await expect(backend.clickTile(1, "fr")).rejects.toBeInstanceOf(SessionUnavailableError)
        expect(click).toHaveBeenCalledTimes(2)
        backend.close()
    })

    it("reports a session that could not be obtained without attempting the click", async () => {
        const click = vi.fn().mockResolvedValue({})
        const backend = backendWith(click, failingSession())

        await expect(backend.clickTile(1, "fr")).rejects.toBeInstanceOf(SessionUnavailableError)
        expect(click).not.toHaveBeenCalled()
        backend.close()
    })

    it("reports the per-IP throttle as a RateLimitedError, without retrying", async () => {
        const refused = new ConnectError("too many clicks", Code.ResourceExhausted)
        const click = vi.fn().mockRejectedValue(refused)
        const backend = backendWith(click)

        await expect(backend.clickTile(1, "fr")).rejects.toBeInstanceOf(RateLimitedError)
        expect(click).toHaveBeenCalledTimes(1)
        backend.close()
    })

    it("keeps the refusal as the cause, so the console still has the detail", async () => {
        const refused = new ConnectError("too many clicks", Code.ResourceExhausted)
        const backend = backendWith(vi.fn().mockRejectedValue(refused))

        await expect(backend.clickTile(1, "fr")).rejects.toMatchObject({cause: refused})
        backend.close()
    })

    it("reports the VPN refusal as a VPNBlockedError, without retrying", async () => {
        const refused = new ConnectError("clicks from VPN addresses are refused", Code.PermissionDenied)
        const click = vi.fn().mockRejectedValue(refused)
        const backend = backendWith(click)

        await expect(backend.clickTile(1, "fr")).rejects.toBeInstanceOf(VPNBlockedError)
        await expect(backend.clickTile(1, "fr")).rejects.not.toBeInstanceOf(RateLimitedError)
        expect(click).toHaveBeenCalledTimes(2)
        backend.close()
    })

    it("keeps the VPN refusal as the cause too", async () => {
        const refused = new ConnectError("clicks from VPN addresses are refused", Code.PermissionDenied)
        const backend = backendWith(vi.fn().mockRejectedValue(refused))

        await expect(backend.clickTile(1, "fr")).rejects.toMatchObject({cause: refused})
        backend.close()
    })

    it("leaves every other failure as it was", async () => {
        const rejected = new ConnectError("nope", Code.InvalidArgument)
        const backend = backendWith(vi.fn().mockRejectedValue(rejected))

        await expect(backend.clickTile(1, "fr")).rejects.toBe(rejected)
        backend.close()
    })

    it("retries a server it could not reach", async () => {
        vi.spyOn(console, "error").mockImplementation(() => {})
        const click = vi.fn()
            .mockRejectedValueOnce(new ConnectError("offline", Code.Unavailable))
            .mockResolvedValue({})
        const backend = backendWith(click)

        await backend.clickTile(1, "fr")
        expect(click).toHaveBeenCalledTimes(2)
        backend.close()
    })
})

describe("PlanetBackend click budget", () => {
    const budgetClient = (
        click: ReturnType<typeof vi.fn>,
        getBudget: ReturnType<typeof vi.fn> = noBudget(),
    ) => ({click, getMap: vi.fn(), getBudget, mapDensity: vi.fn(), listenForEvents: noEvents()} as never)

    const budget = (tokens: number, capacity = 10, refillPerSecond = 1) =>
        new ClickBudgetMessage({tokens, capacity, refillPerSecond})

    const watch = (backend: PlanetBackend) => {
        const seen: number[] = []
        backend.watchClickBudget(b => seen.push(b.tokens))
        return seen
    }

    it("reads the allowance once at load, for a client that has not clicked yet", async () => {
        const getBudget = vi.fn().mockResolvedValue({budget: budget(7)})
        const backend = new PlanetBackend(budgetClient(vi.fn(), getBudget), 1_000)

        await vi.waitFor(() => expect(getBudget).toHaveBeenCalledTimes(1))

        // Subscribing after the read still gets it: a React tree mounts later.
        expect(watch(backend)).toEqual([7])
        backend.close()
    })

    it("re-anchors on what every click answers", async () => {
        const click = vi.fn()
            .mockResolvedValueOnce({budget: budget(6)})
            .mockResolvedValueOnce({budget: budget(5)})
        const backend = new PlanetBackend(budgetClient(click), 1_000)

        const seen = watch(backend)
        await backend.clickTile(1, "fr")
        await backend.clickTile(2, "fr")

        expect(seen.at(-1)).toBe(5)
        backend.close()
    })

    it("takes a click off the counter the moment it is sent, not when it lands", async () => {
        let land: (res: unknown) => void = () => {}
        const click = vi.fn().mockImplementation(() => new Promise(resolve => {
            land = resolve
        }))
        const getBudget = vi.fn().mockResolvedValue({budget: budget(4)})
        const backend = new PlanetBackend(budgetClient(click, getBudget), 1_000)

        await vi.waitFor(() => expect(getBudget).toHaveBeenCalledTimes(1))
        const seen = watch(backend)

        const inFlight = backend.clickTile(1, "fr")
        expect(seen.at(-1)).toBe(3)

        // The click only leaves once the session has answered, a microtask later.
        await vi.waitFor(() => expect(click).toHaveBeenCalledTimes(1))
        land({budget: budget(3)})
        await inFlight

        expect(seen.at(-1)).toBe(3)
        backend.close()
    })

    it("never promises a click the server has already spent", async () => {
        let land: (res: unknown) => void = () => {}
        const click = vi.fn()
            .mockResolvedValueOnce({budget: budget(9)})
            .mockImplementationOnce(() => new Promise(resolve => {
                land = resolve
            }))
        const backend = new PlanetBackend(budgetClient(click), 1_000)

        const seen = watch(backend)
        await backend.clickTile(1, "fr")

        const second = backend.clickTile(2, "fr")
        expect(seen.at(-1)).toBe(8)

        await vi.waitFor(() => expect(click).toHaveBeenCalledTimes(2))
        land({budget: budget(8)})
        await second
        backend.close()
    })

    it("takes the reading off a refusal, which is where it matters most", async () => {
        const refusal = new ConnectError(
            "too many clicks", Code.ResourceExhausted, undefined, [budget(0)])
        const click = vi.fn().mockRejectedValue(refusal)
        const backend = new PlanetBackend(budgetClient(click), 1_000)

        const seen = watch(backend)
        await expect(backend.clickTile(1, "fr")).rejects.toBeInstanceOf(RateLimitedError)

        expect(seen.at(-1)).toBe(0)
        backend.close()
    })

    it("says nothing at all against a server that does not throttle clicks", async () => {
        const click = vi.fn().mockResolvedValue({})
        const backend = new PlanetBackend(budgetClient(click), 1_000)

        const seen = watch(backend)
        await backend.clickTile(1, "fr")

        expect(seen).toEqual([])
        backend.close()
    })

    it("stays quiet against a server too old to know the call", async () => {
        const error = vi.spyOn(console, "error").mockImplementation(() => {})
        const getBudget = vi.fn().mockRejectedValue(new ConnectError("nope", Code.Unimplemented))
        const backend = new PlanetBackend(budgetClient(vi.fn(), getBudget), 1_000)

        await vi.waitFor(() => expect(getBudget).toHaveBeenCalledTimes(1))

        expect(watch(backend)).toEqual([])
        expect(error).not.toHaveBeenCalled()
        backend.close()
    })

    it("narrows back to the plain burst when a caught bonus ends, with no click", async () => {
        vi.useFakeTimers()
        try {
            const getBudget = vi.fn()
                .mockResolvedValueOnce({budget: budget(10)})
                .mockResolvedValueOnce({budget: budget(10)})
            const claimBonus = vi.fn().mockResolvedValue({
                budget: budget(30, 30, 3),
                kind: BonusKind.TRIPLE_CLICKS,
                durationSeconds: 20,
            })
            const client = {...budgetClient(vi.fn(), getBudget) as object, claimBonus} as never
            const backend = new PlanetBackend(client, 1_000)

            const capacities: number[] = []
            backend.watchClickBudget(b => capacities.push(b.capacity))

            await backend.claimBonus("t", "fr")
            expect(capacities.at(-1)).toBe(30)

            await vi.advanceTimersByTimeAsync(20_000)

            expect(getBudget).toHaveBeenCalledTimes(2)
            expect(capacities.at(-1)).toBe(10)
            backend.close()
        } finally {
            vi.useRealTimers()
        }
    })

    it("reads how many shapes an enclose claim is worth, and how big", async () => {
        const claimBonus = vi.fn().mockResolvedValue({
            budget: budget(10),
            kind: BonusKind.ENCLOSE_CLICKS,
            durationSeconds: 30,
            enclosures: 3,
            enclosureMaxTiles: 10,
        })
        const client = {...budgetClient(vi.fn()) as object, claimBonus} as never
        const backend = new PlanetBackend(client, 1_000)

        await expect(backend.claimBonus("t", "fr"))
            .resolves.toEqual({kind: "encloseClicks", seconds: 30, shapes: 3, maxTiles: 10})
        backend.close()
    })

    it("reads the price the server sends, already divided into clicks", async () => {
        const click = vi.fn().mockResolvedValue({
            budget: new ClickBudgetMessage({tokens: 1, capacity: 1, refillPerSecond: 0.125, cost: 8, share: 0.8, nextShare: 0.9, nextCost: 10}),
        })
        const backend = new PlanetBackend(budgetClient(click), 1_000)

        const seen: ClickBudget[] = []
        backend.watchClickBudget(b => seen.push(b))
        await backend.clickTile(1, "bg")

        expect(seen.at(-1)?.capacity).toBe(1)
        expect(seen.at(-1)?.price).toEqual({cost: 8, share: 0.8, next: {share: 0.9, cost: 10}})
        backend.close()
    })

    it("says nothing about price against a server too old to send one", async () => {
        const click = vi.fn().mockResolvedValue({budget: budget(6)})
        const backend = new PlanetBackend(budgetClient(click), 1_000)

        const seen: ClickBudget[] = []
        backend.watchClickBudget(b => seen.push(b))
        await backend.clickTile(1, "fr")

        expect(seen.at(-1)?.price).toBeUndefined()
        backend.close()
    })

    it("re-reads the allowance for the country the player switches to", async () => {
        const getBudget = vi.fn().mockImplementation(({countryId}: {countryId: string}) =>
            Promise.resolve({budget: budget(countryId === "bg" ? 1 : 10)}))
        const backend = new PlanetBackend(budgetClient(vi.fn(), getBudget), 1_000)

        const seen = watch(backend)
        backend.priceFor("bg")

        await vi.waitFor(() => expect(getBudget).toHaveBeenCalledWith({countryId: "bg"}))
        await vi.waitFor(() => expect(seen.at(-1)).toBe(1))

        backend.priceFor("bg")
        expect(getBudget).toHaveBeenCalledTimes(2)
        backend.close()
    })

    it("drops a reading priced for a country the player has left", async () => {
        let land: (res: unknown) => void = () => {}
        const click = vi.fn().mockImplementation(() => new Promise(resolve => {
            land = resolve
        }))
        const getBudget = vi.fn().mockResolvedValue({budget: budget(2)})
        const backend = new PlanetBackend(budgetClient(click, getBudget), 1_000)
        backend.priceFor("bg")
        await vi.waitFor(() => expect(getBudget).toHaveBeenCalledTimes(2))

        const seen = watch(backend)
        const inFlight = backend.clickTile(1, "fr")
        await vi.waitFor(() => expect(click).toHaveBeenCalledTimes(1))
        land({budget: budget(9)})
        await inFlight

        expect(seen).not.toContain(9)
        backend.close()
    })

    it("stops reporting once unsubscribed", async () => {
        const click = vi.fn().mockResolvedValue({budget: budget(6)})
        const backend = new PlanetBackend(budgetClient(click), 1_000)

        const seen: number[] = []
        const stop = backend.watchClickBudget(b => seen.push(b.tokens))
        stop()

        await backend.clickTile(1, "fr")

        expect(seen).toEqual([])
        backend.close()
    })
})

describe("offerOf", () => {
    const offered = (fields: {
        token?: string,
        seed?: number,
        kind?: BonusKind,
        durationSeconds?: number,
        expiresAtUnixMs?: bigint,
    } = {}) => new PlanetEvent({
        event: {
            case: "bonusOffered",
            value: new BonusOffered({
                token: "a-token",
                seed: 42,
                kind: BonusKind.TRIPLE_CLICKS,
                durationSeconds: 60,
                expiresAtUnixMs: 1_000_000n,
                ...fields,
            }),
        },
    })

    it("reads the box the server addressed to this client", () => {
        const offer = offerOf(offered())

        expect(offer?.token).toBe("a-token")
        expect(offer?.seed).toBe(42)
        expect(offer?.reward).toEqual({kind: "tripleClicks", seconds: 60})
    })

    it("builds the deadline from how long is left, not from the server's clock", () => {
        // The two wall clocks are unrelated. Taking the timestamp at face value
        // would make every box look already lapsed on a client running fast.
        const offer = offerOf(offered({expiresAtUnixMs: 1_015_000n}), 5_000, 1_000_000)

        expect(offer?.expiresAt).toBe(5_000 + 15_000)
    })

    it("is unmoved by a client clock that is minutes out", () => {
        const skewed = offerOf(offered({expiresAtUnixMs: 1_015_000n}), 5_000, 1_000_000 + 600_000)
        const honest = offerOf(offered({expiresAtUnixMs: 1_015_000n}), 5_000, 1_000_000)

        expect(honest!.expiresAt - 5_000).toBe(15_000)
        // The skew is carried, but the box is still given its full window
        // relative to the reading rather than being born expired.
        expect(skewed!.expiresAt).toBeLessThan(honest!.expiresAt)
    })

    it("reads a bomb box as one, with no blast radius until it is claimed", () => {
        expect(offerOf(offered({kind: BonusKind.BOMB, durationSeconds: 30}))?.reward)
            .toEqual({kind: "bomb", seconds: 30, radius: 0})
    })

    it("reads an enclose box as one", () => {
        expect(offerOf(offered({kind: BonusKind.ENCLOSE_CLICKS}))?.reward.kind).toBe("encloseClicks")
    })

    it("reads a spread box as one", () => {
        expect(offerOf(offered({kind: BonusKind.SPREAD_CLICKS}))?.reward)
            .toEqual({kind: "spreadClicks", seconds: 60})
    })

    it("drops a kind this build cannot describe rather than guessing at it", () => {
        expect(offerOf(offered({kind: BonusKind.UNSPECIFIED}))).toBeUndefined()
    })

    it("drops everything that is not an offer", () => {
        expect(offerOf(new PlanetEvent({event: {case: "heartbeat", value: new Heartbeat()}}))).toBeUndefined()
        expect(offerOf(tileUpdateEvent({tileId: 1, countryId: "fr"}))).toBeUndefined()
    })
})

describe("catchOf", () => {
    it("reads who caught one", () => {
        const event = new PlanetEvent({
            event: {case: "bonusTaken", value: new BonusTaken({countryId: "jp"})},
        })

        expect(catchOf(event)).toEqual({countryId: "jp"})
    })

    it("drops everything that is not a catch", () => {
        expect(catchOf(new PlanetEvent({event: {case: "heartbeat", value: new Heartbeat()}}))).toBeUndefined()
    })
})

describe("enclosureOf", () => {
    const enclosed = (fields: Partial<TilesEnclosed> = {}) => new PlanetEvent({
        event: {
            case: "tilesEnclosed",
            value: new TilesEnclosed({
                countryId: "jp",
                closingTileId: 4,
                wallTileIds: [4, 5, 6],
                filledTileIds: [9, 10],
                ...fields,
            }),
        },
    })

    it("reads the shape and what it took", () => {
        expect(enclosureOf(enclosed())).toEqual({
            countryId: "jp",
            closingTile: 4,
            wall: [4, 5, 6],
            filled: [9, 10],
            yours: undefined,
        })
    })

    it("says how many shapes are left only when the shape is this client's", () => {
        expect(enclosureOf(enclosed({yours: true, enclosuresLeft: 0}))?.yours).toEqual({shapesLeft: 0})
        expect(enclosureOf(enclosed({yours: false}))?.yours).toBeUndefined()
    })

    it("drops everything that is not a closed shape", () => {
        expect(enclosureOf(new PlanetEvent({event: {case: "heartbeat", value: new Heartbeat()}}))).toBeUndefined()
    })
})

describe("spreadOf", () => {
    it("reads the tile clicked and the tiles it spread onto", () => {
        const event = new PlanetEvent({
            event: {case: "tilesSpread", value: new TilesSpread({countryId: "br", tileId: 100, spreadTileIds: [99, 101]})},
        })

        expect(spreadOf(event)).toEqual({countryId: "br", tile: 100, spread: [99, 101]})
    })

    it("drops everything that is not a spread click", () => {
        expect(spreadOf(new PlanetEvent({event: {case: "heartbeat", value: new Heartbeat()}}))).toBeUndefined()
    })
})

describe("updateOf with the bonus cases on the stream", () => {
    it("still ignores them, so an old client is unaffected by either", () => {
        const offer = new PlanetEvent({
            event: {case: "bonusOffered", value: new BonusOffered({token: "t"})},
        })

        expect(updateOf(offer)).toBeUndefined()
    })
})

describe("bombOf", () => {
    const dropped = (fields: Partial<{tileId: number, clearedTileIds: number[]}> = {}) => new PlanetEvent({
        event: {
            case: "bombDropped",
            value: new BombDropped({
                tileId: 7,
                countryId: "fr",
                radius: 0.03,
                clearedTileIds: [6, 7, 8],
                point: new GlobePoint({x: 0, y: 1, z: 0}),
                ...fields,
            }),
        },
    })

    it("reads a blast on land, with every tile it cleared", () => {
        expect(bombOf(dropped())).toEqual({
            tile: 7,
            point: {x: 0, y: 1, z: 0},
            countryId: "fr",
            radius: 0.03,
            cleared: [6, 7, 8],
        })
    })

    it("reads tile 0 as a bomb that fell in the sea", () => {
        const splash = bombOf(dropped({tileId: 0, clearedTileIds: []}))

        expect(splash?.tile).toBeUndefined()
        expect(splash?.cleared).toEqual([])
    })

    it("drops everything that is not a blast, and a tile update is not one", () => {
        expect(bombOf(tileUpdateEvent({tileId: 1, countryId: "fr"}))).toBeUndefined()
        expect(updateOf(dropped())).toBeUndefined()
    })
})

describe("PlanetBackend bombs", () => {
    const clientWith = (fields: Record<string, unknown>) =>
        ({click: vi.fn(), getMap: vi.fn(), getBudget: noBudget(), mapDensity: vi.fn(), listenForEvents: noEvents(), ...fields}) as never

    it("sends where the player aimed and the session token", async () => {
        const dropBomb = vi.fn().mockResolvedValue({})
        const backend = new PlanetBackend(clientWith({dropBomb}), 1_000, fixedSession("session-1"))

        await backend.dropBomb({x: 1, y: 2, z: 3}, "fr")

        expect(dropBomb).toHaveBeenCalledWith({target: {x: 1, y: 2, z: 3}, countryId: "fr"}, expect.anything())
        expect((dropBomb.mock.calls[0][1] as {headers: Headers}).headers.get(SESSION_HEADER)).toBe("session-1")
        backend.close()
    })

    it("reports a bomb the server no longer holds for this player as lost", async () => {
        const dropBomb = vi.fn().mockRejectedValue(new ConnectError("gone", Code.NotFound))
        const backend = new PlanetBackend(clientWith({dropBomb}), 1_000)

        await expect(backend.dropBomb({x: 1, y: 0, z: 0}, "fr")).rejects.toBeInstanceOf(BonusLostError)
        backend.close()
    })

    it("hands over the tile updates that came before a blast, before the blast", async () => {
        const events = [
            tileUpdateEvent({tileId: 7, countryId: "jp"}),
            new PlanetEvent({
                event: {case: "bombDropped", value: new BombDropped({tileId: 7, countryId: "fr", clearedTileIds: [7]})},
            }),
        ]
        const listenForEvents = vi.fn(async function* () {
            yield* events
            await new Promise(() => {})
        })
        const backend = new PlanetBackend(clientWith({listenForEvents}), 60_000)

        const order: string[] = []
        backend.listenForUpdatesBatch(() => order.push("updates"))
        backend.listenForBombs(() => order.push("bomb"))

        await vi.waitFor(() => expect(order).toEqual(["updates", "bomb"]))
        backend.close()
    })
})

describe("asBonusError", () => {
    it("reports a box that is gone as lost, whichever way the server said so", () => {
        expect(asBonusError(new ConnectError("gone", Code.NotFound))).toBeInstanceOf(BonusLostError)
        expect(asBonusError(new ConnectError("off", Code.Unimplemented))).toBeInstanceOf(BonusLostError)
    })

    it("keeps a session failure as one, since that is the player's clicks stopping too", () => {
        expect(asBonusError(new ConnectError("no", Code.Unauthenticated)))
            .toBeInstanceOf(SessionUnavailableError)
    })

    it("leaves anything else alone", () => {
        const boom = new Error("boom")
        expect(asBonusError(boom)).toBe(boom)
    })
})
