import {describe, expect, it, vi} from "vitest"
import {
    asBonusError,
    bindingsOf,
    bombOf,
    countsOf,
    fortificationOf,
    knockOutOf,
    cutOffOf,
    shieldsOf,
    enclosureOf,
    takenOf,
    PlanetBackend,
    spreadOf,
    updateOf,
    virusSpreadOf,
    minesExplodedOf,
    mineLaidOf,
    rallyCalledOf,
} from "./planetBackend.ts"
import {Code, ConnectError} from "@connectrpc/connect"
import {BoardChanges, CACHED_MS} from "./boards.ts"
import {
    Allegiance as AllegianceMessage,
    BanRefusal,
    BombRefusal,
    FrozenRefusal,
    BombDropped,
    BonusKind,
    ChargeGained,
    ChargesGranted,
    ChargesHeld,
    ClickBudget as ClickBudgetMessage,
    GetAllegianceResponse,
    PledgeAllegianceResponse,
    GetChargeRulesResponse,
    GetHoldingsResponse,
    GetMapResponse,
    GetLandmassResponse,
    GetProvinceResponse,
    GetFortressesResponse,
    Fortress as FortressMessage,
    Holding,
    HoldingsChanged,
    LandmassHoldings,
    ProvinceHoldings,
    MapVersion as MapVersionMessage,
    GetTileHolderResponse,
    ProvinceFortification as ProvinceFortificationMessage,
    LandmassHolder as LandmassHolderMessage,
    GlobePoint,
    Heartbeat,
    KnockOut as KnockOutMessage,
    CutOff as CutOffMessage,
    ProvinceFortified,
    PlanetEvent,
    ReachRefusal,
    RulesChanged,
    SharedWith,
    TilesEnclosed,
    TileHolder as TileHolderMessage,
    TileShields,
    TilesSpread,
    TileUpdate,
    InfectedTile,
    VirusRefusal,
    VirusSpread as VirusSpreadMessage,
    GetMinesResponse,
    GetTrapsResponse,
    GetRalliesResponse,
    CallRallyResponse,
    Rally as RallyMessage,
    RallyRefusal,
    MineBlast as MineBlastMessage,
    MineLaid as MineLaidMessage,
    MineRefusal,
    TrapRefusal,
    TrapSprung as TrapSprungMessage,
    MinesExploded as MinesExplodedMessage,
} from "../gen/grpc/planet/v1/planet_pb.ts"
import {AllegianceLockedError, BankFullError, BannedError, BombRefusedError, BonusLostError, FrozenError, MineRefusedError, TrapRefusedError, Minefield, Trapline, TrapSprung, OutOfReachError, RallyRefusedError, ShieldRefusedError, RateLimitedError, VirusRefusedError, VPNBlockedError} from "./backend.ts"
import {GetGrantRulesResponse} from "../gen/grpc/bonuses/v1/bonuses_pb.ts"
import {NoSession, SESSION_HEADER, type SessionProvider, SessionUnavailableError} from "./session.ts"
import type {ClickBudget} from "./clickBudget.ts"
import type {BonusReward, BonusRules, Charges} from "../domain/bonus.ts"
import {ALL_OFF} from "../domain/bonus.ts"
import type {Rally} from "../domain/rallies.ts"
import type {Allegiance} from "../domain/allegiance.ts"
import {tileRanges} from "../domain/mapTiles.ts"
import type {Counts, MapVersion} from "./backend.ts"

const NO_VERSION = {epoch: 0, seq: 0}

function fixedSession(token: string): SessionProvider {
    return {token: async () => token, held: () => token, identity: async () => token, heldIdentity: () => token, invalidate: () => {}}
}

function sessionMinting(token: string): SessionProvider {
    let minted: string | undefined
    return {
        token: async () => (minted = token),
        held: () => minted,
        identity: async () => minted,
        heldIdentity: () => minted,
        invalidate: () => {
            minted = undefined
        },
    }
}

function rotatingSession(tokens: string[]) {
    let index = 0
    return {
        invalidated: 0,
        async token() {
            return tokens[Math.min(index, tokens.length - 1)]
        },
        held() {
            return tokens[Math.min(index, tokens.length - 1)]
        },
        async identity() {
            return tokens[Math.min(index, tokens.length - 1)]
        },
        heldIdentity() {
            return tokens[Math.min(index, tokens.length - 1)]
        },
        invalidate() {
            this.invalidated++
            index++
        },
    }
}

function renewingSession(first: string) {
    let current = first
    const listeners = new Set<(token: string) => void>()
    return {
        token: async () => current, held: () => current, identity: async () => current, heldIdentity: () => current,
        invalidate: () => {},
        listenForTokens(onToken: (token: string) => void) {
            listeners.add(onToken)
            return () => listeners.delete(onToken)
        },
        renew(next: string) {
            current = next
            listeners.forEach((listener) => listener(next))
        },
    }
}

function failingSession(): SessionProvider {
    return {
        token: async () => {
            throw new SessionUnavailableError()
        },
        held: () => undefined, identity: async () => undefined, heldIdentity: () => undefined,
        invalidate: () => {},
    }
}

function tileUpdateEvent(fields: {tileId: number, countryId: string, previousCountryId?: string, clicked?: boolean, shields?: number, version?: MapVersionMessage}): PlanetEvent {
    return new PlanetEvent({event: {case: "tileUpdate", value: new TileUpdate(fields)}})
}

describe("updateOf", () => {
    it("maps a tile update onto the shape the globe consumes", () => {
        expect(updateOf(tileUpdateEvent({tileId: 7, countryId: "jp", previousCountryId: "fr"})))
            .toEqual({tile: 7, previousCountry: "fr", newCountry: "jp", clicked: false, shields: 0, version: NO_VERSION})
    })

    it("reports an unowned previous tile as undefined rather than an empty code", () => {
        expect(updateOf(tileUpdateEvent({tileId: 1, countryId: "fr"})))
            .toEqual({tile: 1, previousCountry: undefined, newCountry: "fr", clicked: false, shields: 0, version: NO_VERSION})
    })

    it("reports a tile given back to nobody as undefined, so nobody gets a leaderboard row", () => {
        expect(updateOf(tileUpdateEvent({tileId: 1, countryId: "", previousCountryId: "ps"})))
            .toEqual({tile: 1, previousCountry: "ps", newCountry: undefined, clicked: false, shields: 0, version: NO_VERSION})
    })

    it("says which update a click made, so the globe can animate it", () => {
        expect(updateOf(tileUpdateEvent({tileId: 7, countryId: "jp", previousCountryId: "fr", clicked: true}))?.clicked).toBe(true)
    })

    it("says how many shields stand on the tile after the update", () => {
        expect(updateOf(tileUpdateEvent({tileId: 7, countryId: "fr", previousCountryId: "fr", shields: 3}))?.shields).toBe(3)
    })

    it("says the version of the map the update made", () => {
        const version = new MapVersionMessage({epoch: 1_790_000_000, seq: BigInt(42)})

        expect(updateOf(tileUpdateEvent({tileId: 7, countryId: "fr", version}))?.version).toEqual({epoch: 1_790_000_000, seq: 42})
    })

    it("drops a heartbeat", () => {
        const heartbeat = new PlanetEvent({event: {case: "heartbeat", value: new Heartbeat()}})
        expect(updateOf(heartbeat)).toBeUndefined()
    })

    it("drops an event case this build does not know", () => {
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

function noBudget() {
    return vi.fn().mockResolvedValue({})
}

function noEvents() {
    return vi.fn(async function* () {})
}


function bonusesWith(fields: Record<string, unknown> = {}) {
    return {
        getGrantRules: vi.fn().mockResolvedValue(new GetGrantRulesResponse()),
        ...fields,
    } as never
}

function bonusReads() {
    return {
        getChargeRules: vi.fn().mockResolvedValue(new GetChargeRulesResponse({blastRadius: 0.03, enclosureMaxTiles: 25, spreadClicks: 8, enclosures: 3})),
        getCharges: vi.fn().mockResolvedValue({charges: new ChargesHeld()}),
        getAllegiance: vi.fn().mockResolvedValue(new GetAllegianceResponse({allegiance: new AllegianceMessage({countryId: "fr"})})),
        pledgeAllegiance: vi.fn().mockResolvedValue(new PledgeAllegianceResponse()),
        getMines: vi.fn().mockResolvedValue(new GetMinesResponse()),
        getTraps: vi.fn().mockResolvedValue(new GetTrapsResponse()),
        getRallies: vi.fn().mockResolvedValue(new GetRalliesResponse()),
    }
}

type GetMapRequestFields = {startTileId: number, endTileId: number}
type GetMapMock = ReturnType<typeof getMapMock>

function getMapMock(impl?: (req: GetMapRequestFields) => Promise<GetMapResponse>) {
    return vi.fn<(req: GetMapRequestFields) => Promise<GetMapResponse>>(impl)
}

describe("shieldsOf", () => {
    it("reads the shields of the batch's shielded tiles", () => {
        const res = new GetMapResponse({
            startTileId: 1,
            shields: [new TileShields({tileId: 2, shields: 4}), new TileShields({tileId: 9, shields: 10})],
        })

        expect(shieldsOf(res)).toEqual(new Map([[2, 4], [9, 10]]))
    })

    it("reads none from a batch with no shields", () => {
        expect(shieldsOf(mapResponse(1, ["", "fr"], [1]))).toEqual(new Map())
    })
})

describe("PlanetBackend.getOwnerships", () => {
    const backendWith = (getMap: GetMapMock) => {
        const client = {click: vi.fn(), getMap, getBudget: noBudget(), ...bonusReads(), mapDensity: vi.fn(), listenForEvents: noEvents()} as never
        return new PlanetBackend(client, 1_000)
    }

    const collect = async (backend: PlanetBackend, signal?: AbortSignal) => {
        const seen: Map<number, string>[] = []
        await backend.getOwnerships(tileRanges(4, 2), o => seen.push(o.bindings), signal)
        return seen
    }

    it("hands each batch its shields beside its owners, with its range and the version it was read at", async () => {
        const getMap = getMapMock(async () => new GetMapResponse({
            startTileId: 1, codes: ["", "fr"], tiles: tileBytes([1, 1]),
            shields: [new TileShields({tileId: 2, shields: 3})],
            version: new MapVersionMessage({epoch: 1_790_000_000, seq: BigInt(9)}),
        }))
        const backend = backendWith(getMap)

        const seen: unknown[] = []
        await backend.getOwnerships([{start: 1, end: 2}], ({range, version, shields}) => seen.push({range, version, shields}))

        expect(seen).toEqual([{range: {start: 1, end: 2}, version: {epoch: 1_790_000_000, seq: 9}, shields: new Map([[2, 3]])}])
        backend.close()
    })

    it("says which flags are knocked out", async () => {
        const getMap = getMapMock(async () => new GetMapResponse({
            startTileId: 1, codes: ["", "fr"], tiles: tileBytes([1, 0]), knockedOut: ["de", "it"],
            version: new MapVersionMessage({epoch: 1_790_000_000, seq: BigInt(9)}),
        }))
        const backend = backendWith(getMap)

        const seen: (readonly string[])[] = []
        await backend.getOwnerships([{start: 1, end: 2}], ({knockedOut}) => seen.push(knockedOut))

        expect(seen).toEqual([["de", "it"]])
        backend.close()
    })

    it("reads each tile once, from the first to the last", async () => {
        const getMap = getMapMock(async () => mapResponse(1, ["", "fr"], [1, 0]))
        const backend = backendWith(getMap)

        await backend.getOwnerships(tileRanges(5, 2), () => {})

        expect(getMap.mock.calls.map(c => c[0])).toEqual([
            {startTileId: 1, endTileId: 2},
            {startTileId: 3, endTileId: 4},
            {startTileId: 5, endTileId: 5},
        ])
        backend.close()
    })

    it("keeps eight reads in flight", async () => {
        let held = true
        const answers: (() => void)[] = []
        const getMap = getMapMock((req) => {
            const response = mapResponse(req.startTileId, [""], [0])
            if (!held) return Promise.resolve(response)
            return new Promise((resolve) => answers.push(() => resolve(response)))
        })
        const backend = backendWith(getMap)

        const loading = backend.getOwnerships(tileRanges(20, 1), () => {})
        await vi.waitFor(() => expect(getMap).toHaveBeenCalledTimes(8))
        answers[0]()
        await vi.waitFor(() => expect(getMap).toHaveBeenCalledTimes(9))

        held = false
        answers.forEach((answer) => answer())
        await loading
        expect(getMap).toHaveBeenCalledTimes(20)
        backend.close()
    })

    it("hands each batch over as it lands", async () => {
        const answers = new Map<number, () => void>()
        const getMap = getMapMock((req) => new Promise((resolve) => {
            answers.set(req.startTileId, () => resolve(mapResponse(req.startTileId, ["", "fr"], [1, 1])))
        }))
        const backend = backendWith(getMap)

        const seen: Map<number, string>[] = []
        const loading = backend.getOwnerships(tileRanges(4, 2), o => seen.push(o.bindings))
        await vi.waitFor(() => expect(answers.size).toBe(2))
        answers.get(3)!()
        answers.get(1)!()
        await loading

        expect(seen).toEqual([new Map([[3, "fr"], [4, "fr"]]), new Map([[1, "fr"], [2, "fr"]])])
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

        await expect(backend.getOwnerships(tileRanges(4, 4), () => {})).rejects.toThrow(/nope/)
        expect(getMap).toHaveBeenCalledTimes(1)
        backend.close()
    })

    it("stops the other reads when one fails", async () => {
        const signals: AbortSignal[] = []
        const getMap = vi.fn((req: GetMapRequestFields, {signal}: {signal: AbortSignal}) => {
            signals.push(signal)
            if (req.startTileId === 1) return Promise.reject(new ConnectError("nope", Code.InvalidArgument))
            return new Promise<GetMapResponse>((_, reject) => signal.addEventListener("abort", () => reject(signal.reason)))
        })
        const backend = backendWith(getMap as never)

        await expect(backend.getOwnerships(tileRanges(20, 1), () => {})).rejects.toThrow(/nope/)
        expect(getMap.mock.calls.length).toBeLessThanOrEqual(8)
        expect(signals.every((signal) => signal.aborted)).toBe(true)
        backend.close()
    })

    it("hands over nothing that lands after the caller aborted", async () => {
        const controller = new AbortController()
        let answer = () => {}
        const getMap = getMapMock(() => new Promise((resolve) => {
            answer = () => resolve(mapResponse(1, ["", "fr"], [1]))
        }))
        const backend = backendWith(getMap)

        const seen: Map<number, string>[] = []
        const loading = backend.getOwnerships(tileRanges(1, 1), o => seen.push(o.bindings), controller.signal)
        await vi.waitFor(() => expect(getMap).toHaveBeenCalledTimes(1))
        controller.abort()
        answer()

        await expect(loading).rejects.toThrow()
        expect(seen).toEqual([])
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
        const clientStub = {click, getMap: vi.fn(), getBudget: noBudget(), ...bonusReads(), mapDensity: vi.fn(), listenForEvents: noEvents()} as never
        return new PlanetBackend(clientStub, 1_000, session)
    }

    const headersOf = (click: ReturnType<typeof vi.fn>, call = 0) =>
        (click.mock.calls[call][1] as {headers: Headers}).headers

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

    it("reports a click on a frozen map as a FrozenError", async () => {
        const refused = new ConnectError("the map is frozen", Code.FailedPrecondition, undefined, [new FrozenRefusal()])
        const click = vi.fn().mockRejectedValue(refused)
        const backend = backendWith(click)

        await expect(backend.clickTile(1, "fr")).rejects.toBeInstanceOf(FrozenError)
        expect(click).toHaveBeenCalledTimes(1)
        backend.close()
    })

    it("reports a click out of the flag's reach as an OutOfReachError, sent once", async () => {
        const refused = new ConnectError("out of reach", Code.FailedPrecondition, undefined, [new ReachRefusal()])
        const click = vi.fn().mockRejectedValue(refused)
        const backend = backendWith(click)

        await expect(backend.clickTile(1, "fr")).rejects.toBeInstanceOf(OutOfReachError)
        expect(click).toHaveBeenCalledTimes(1)
        backend.close()
    })

    it("says when the rules freeze the map", async () => {
        const click = vi.fn()
            .mockResolvedValueOnce({budget: new ClickBudgetMessage({tokens: 6, capacity: 10, refillPerSecond: 1, frozen: true})})
            .mockResolvedValueOnce({budget: new ClickBudgetMessage({tokens: 6, capacity: 10, refillPerSecond: 1})})
        const backend = backendWith(click)

        const seen: (boolean | undefined)[] = []
        backend.watchClickBudget(b => seen.push(b.frozen))
        await backend.clickTile(1, "fr")
        expect(seen.at(-1)).toBe(true)
        await backend.clickTile(2, "fr")
        expect(seen.at(-1)).toBe(false)

        backend.close()
    })

    it("reports a ban as a BannedError with its end, never as the VPN", async () => {
        const until = 1_700_003_600_000
        const refused = new ConnectError("the caller is banned", Code.PermissionDenied, undefined,
            [new BanRefusal({bannedUntilUnixMs: BigInt(until)})])
        const click = vi.fn().mockRejectedValue(refused)
        const backend = backendWith(click)

        const error = await backend.clickTile(1, "fr").catch((e: unknown) => e)

        expect(error).toBeInstanceOf(BannedError)
        expect(error).toMatchObject({until, cause: refused})
        expect(click).toHaveBeenCalledTimes(1)
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

    it("does not retry a click that timed out, which may have landed", async () => {
        const click = vi.fn().mockRejectedValue(new ConnectError("the operation timed out", Code.DeadlineExceeded))
        const backend = backendWith(click)

        await expect(backend.clickTile(1, "fr")).rejects.toThrow(/timed out/)
        expect(click).toHaveBeenCalledTimes(1)
        backend.close()
    })
})

describe("PlanetBackend click budget", () => {
    const budgetClient = (
        click: ReturnType<typeof vi.fn>,
        getBudget: ReturnType<typeof vi.fn> = noBudget(),
    ) => ({click, getMap: vi.fn(), getBudget, ...bonusReads(), mapDensity: vi.fn(), listenForEvents: noEvents()} as never)

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

        expect(watch(backend)).toEqual([7])
        backend.close()
    })

    it("reads the allowance again when the rules change, so the meter follows the new pace", async () => {
        const getBudget = vi.fn()
            .mockResolvedValueOnce({budget: budget(4, 10, 0.2)})
            .mockResolvedValueOnce({budget: budget(4, 10, 0.6)})
        let changeRules = () => {}
        const listenForEvents = vi.fn(async function* () {
            yield new PlanetEvent({event: {case: "heartbeat", value: new Heartbeat()}})
            await new Promise<void>(resolve => changeRules = resolve)
            yield new PlanetEvent({event: {case: "rulesChanged", value: new RulesChanged()}})
            await new Promise(() => {})
        })
        const backend = new PlanetBackend(
            {click: vi.fn(), getMap: vi.fn(), getBudget, ...bonusReads(), mapDensity: vi.fn(), listenForEvents} as never,
            1_000)
        const rates: number[] = []
        backend.watchClickBudget(b => rates.push(b.perSecond))
        await vi.waitFor(() => expect(rates).toEqual([0.2]))

        changeRules()

        await vi.waitFor(() => expect(rates).toEqual([0.2, 0.6]))
        backend.close()
    })

    it("reads the allowance and the charge rules again on a stream that comes back, never on its first", async () => {
        const getBudget = vi.fn()
            .mockResolvedValueOnce({budget: budget(4, 10, 0.2)})
            .mockResolvedValueOnce({budget: budget(4, 10, 0.6)})
        const reads = bonusReads()
        reads.getChargeRules
            .mockResolvedValueOnce(new GetChargeRulesResponse({enclosureMaxTiles: 25}))
            .mockResolvedValueOnce(new GetChargeRulesResponse({enclosureMaxTiles: 35}))
        const heartbeat = new PlanetEvent({event: {case: "heartbeat", value: new Heartbeat()}})
        let drop = () => {}
        const listenForEvents = vi.fn()
            .mockImplementationOnce(async function* () {
                yield heartbeat
                await new Promise<void>(resolve => drop = resolve)
            })
            .mockImplementationOnce(async function* () {
                yield heartbeat
                await new Promise(() => {})
            })
        const backend = new PlanetBackend(
            {click: vi.fn(), getMap: vi.fn(), getBudget, ...reads, mapDensity: vi.fn(), listenForEvents} as never,
            1_000)
        const rates: number[] = []
        backend.watchClickBudget(b => rates.push(b.perSecond))
        const sizes: number[] = []
        const stop = backend.listenForBonuses({
            onVirusSpread: () => {}, onMines: () => {}, onMineLaid: () => {}, onTraps: () => {}, onTrapLaid: () => {}, onTrapSprung: () => {}, onMinesExploded: () => {},
            onRallies: () => {},
            onRallyCalled: () => {}, onEnclosed: () => {}, onSpread: () => {}, onCharges: () => {},
            onRules: rules => sizes.push(rules.enclosureMaxTiles),
        })
        await vi.waitFor(() => expect(rates).toEqual([0.2]))
        expect(reads.getChargeRules).toHaveBeenCalledOnce()

        drop()

        await vi.waitFor(() => expect(rates).toEqual([0.2, 0.6]), {timeout: 3_000})
        await vi.waitFor(() => expect(sizes.at(-1)).toBe(35))
        expect(listenForEvents).toHaveBeenCalledTimes(2)
        stop()
        backend.close()
    })

    it("says who else spends from the bucket, and nothing for the player's own", async () => {
        const click = vi.fn()
            .mockResolvedValueOnce({budget: new ClickBudgetMessage({tokens: 6, capacity: 10, refillPerSecond: 1, sharedWith: SharedWith.GUESTS})})
            .mockResolvedValueOnce({budget: new ClickBudgetMessage({tokens: 5, capacity: 10, refillPerSecond: 1, sharedWith: SharedWith.NETWORK})})
            .mockResolvedValueOnce({budget: new ClickBudgetMessage({tokens: 4, capacity: 10, refillPerSecond: 1, sharedWith: SharedWith.NOBODY})})
            .mockResolvedValueOnce({budget: budget(3)})
        const backend = new PlanetBackend(budgetClient(click), 1_000)

        const seen: (string | undefined)[] = []
        backend.watchClickBudget(b => seen.push(b.sharedWith))
        await backend.clickTile(1, "fr")
        expect(seen.at(-1)).toBe("guests")
        await backend.clickTile(2, "fr")
        expect(seen.at(-1)).toBe("network")
        await backend.clickTile(3, "fr")
        expect(seen.at(-1)).toBeUndefined()
        await backend.clickTile(4, "fr")
        expect(seen.at(-1)).toBeUndefined()

        backend.close()
    })

    it("says what signing in multiplies the allowance by, and nothing from a server that grants nothing", async () => {
        const click = vi.fn()
            .mockResolvedValueOnce({budget: new ClickBudgetMessage({tokens: 6, capacity: 10, refillPerSecond: 1, linkedMultiplier: 2})})
            .mockResolvedValueOnce({budget: new ClickBudgetMessage({tokens: 5, capacity: 10, refillPerSecond: 1, linkedMultiplier: 1})})
            .mockResolvedValueOnce({budget: budget(4)})
        const backend = new PlanetBackend(budgetClient(click), 1_000)

        const seen: (number | undefined)[] = []
        backend.watchClickBudget(b => seen.push(b.linkedMultiplier))
        await backend.clickTile(1, "fr")
        expect(seen.at(-1)).toBe(2)
        await backend.clickTile(2, "fr")
        expect(seen.at(-1)).toBeUndefined()
        await backend.clickTile(3, "fr")
        expect(seen.at(-1)).toBeUndefined()

        backend.close()
    })

    it("says while a guest refills at the signed-in pace of its fast start", async () => {
        const click = vi.fn()
            .mockResolvedValueOnce({budget: new ClickBudgetMessage({tokens: 6, capacity: 10, refillPerSecond: 2, fastStart: true})})
            .mockResolvedValueOnce({budget: new ClickBudgetMessage({tokens: 5, capacity: 10, refillPerSecond: 1})})
        const backend = new PlanetBackend(budgetClient(click), 1_000)

        const seen: (boolean | undefined)[] = []
        backend.watchClickBudget(b => seen.push(b.fastStart))
        await backend.clickTile(1, "fr")
        expect(seen.at(-1)).toBe(true)
        await backend.clickTile(2, "fr")
        expect(seen.at(-1)).toBe(false)

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

    it("reads the price the server sends", async () => {
        const click = vi.fn().mockResolvedValue({
            budget: new ClickBudgetMessage({tokens: 1, capacity: 10, refillPerSecond: 0.125, slowdown: 8, share: 0.8, nextShare: 0.9, nextSlowdown: 10}),
        })
        const backend = new PlanetBackend(budgetClient(click), 1_000)

        const seen: ClickBudget[] = []
        backend.watchClickBudget(b => seen.push(b))
        await backend.clickTile(1, "bg")

        expect(seen.at(-1)?.capacity).toBe(10)
        expect(seen.at(-1)?.price).toEqual({slowdown: 8, share: 0.8, next: {share: 0.9, slowdown: 10}})
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

    it("re-reads the price for the country the player switches to", async () => {
        const getBudget = vi.fn().mockImplementation(({countryId}: {countryId: string}) =>
            Promise.resolve({budget: new ClickBudgetMessage({tokens: 5, capacity: 10, refillPerSecond: 1, slowdown: countryId === "bg" ? 2 : 1})}))
        const backend = new PlanetBackend(budgetClient(vi.fn(), getBudget), 1_000)

        const seen: ClickBudget[] = []
        backend.watchClickBudget(b => seen.push(b))
        backend.priceFor("bg")

        await vi.waitFor(() => expect(getBudget).toHaveBeenCalledWith({countryId: "bg"}, expect.anything()))
        await vi.waitFor(() => expect(seen.at(-1)?.price?.slowdown).toBe(2))
        expect(seen.at(-1)?.tokens).toBe(5)

        backend.priceFor("bg")
        expect(getBudget).toHaveBeenCalledTimes(2)
        backend.close()
    })

    it("reads the allowance with the token it holds, and without one when it holds none", async () => {
        const getBudget = vi.fn().mockResolvedValue({budget: budget(5)})
        const held = new PlanetBackend(budgetClient(vi.fn(), getBudget), 1_000, fixedSession("session-1"))
        await vi.waitFor(() => expect(getBudget).toHaveBeenCalledTimes(1))
        expect((getBudget.mock.calls[0][1] as {headers: Headers}).headers.get(SESSION_HEADER)).toBe("session-1")
        held.close()

        const none = new PlanetBackend(budgetClient(vi.fn(), getBudget), 1_000, failingSession())
        await vi.waitFor(() => expect(getBudget).toHaveBeenCalledTimes(2))
        expect((getBudget.mock.calls[1][1] as {headers: Headers}).headers.get(SESSION_HEADER)).toBeNull()
        none.close()
    })

    it("keeps the count of a reading for a country the player has left, and not its price", async () => {
        let land: (res: unknown) => void = () => {}
        const click = vi.fn().mockImplementation(() => new Promise(resolve => {
            land = resolve
        }))
        const getBudget = vi.fn().mockResolvedValue({budget: new ClickBudgetMessage({tokens: 2, capacity: 10, refillPerSecond: 1, slowdown: 2})})
        const backend = new PlanetBackend(budgetClient(click, getBudget), 1_000)
        backend.priceFor("bg")
        await vi.waitFor(() => expect(getBudget).toHaveBeenCalledTimes(2))

        const seen: ClickBudget[] = []
        backend.watchClickBudget(b => seen.push(b))
        const inFlight = backend.clickTile(1, "fr")
        await vi.waitFor(() => expect(click).toHaveBeenCalledTimes(1))
        land({budget: new ClickBudgetMessage({tokens: 9, capacity: 10, refillPerSecond: 1, slowdown: 1})})
        await inFlight

        expect(seen.at(-1)?.tokens).toBe(9)
        expect(seen.at(-1)?.price?.slowdown).toBe(2)
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

    it("reads the shape, what it filled, what it struck and its version", () => {
        expect(enclosureOf(enclosed({struck: [new TileShields({tileId: 10, shields: 2})], version: versionAt(42)}))).toEqual({
            countryId: "jp",
            closingTile: 4,
            wall: [4, 5, 6],
            filled: [9, 10],
            struck: [{tile: 10, shields: 2}],
            version: {epoch: versionAt(42).epoch, seq: 42},
            yours: undefined,
        })
        expect(takenOf(enclosureOf(enclosed({struck: [new TileShields({tileId: 10, shields: 2})]}))!)).toEqual([9])
    })

    it("says when the shape is this client's", () => {
        expect(enclosureOf(enclosed({yours: true}))?.yours).toBe(true)
        expect(enclosureOf(enclosed({yours: false}))?.yours).toBeUndefined()
    })

    it("drops everything that is not a closed shape", () => {
        expect(enclosureOf(new PlanetEvent({event: {case: "heartbeat", value: new Heartbeat()}}))).toBeUndefined()
    })
})

describe("virusSpreadOf", () => {
    it("reads the virus, its two flags, what it took and struck, its end and its version", () => {
        const event = new PlanetEvent({event: {case: "virusSpread", value: new VirusSpreadMessage({
            virusId: "v-1", countryId: "fr", victimCountryId: "de", takenTileIds: [4, 5],
            struck: [new TileShields({tileId: 6, shields: 1})], ended: true, version: versionAt(43),
        })}})

        expect(virusSpreadOf(event)).toEqual({
            virus: "v-1", countryId: "fr", victimCountryId: "de", taken: [4, 5], struck: [{tile: 6, shields: 1}], ended: true,
            version: {epoch: versionAt(43).epoch, seq: 43},
        })
        expect(virusSpreadOf(new PlanetEvent({event: {case: "heartbeat", value: new Heartbeat()}}))).toBeUndefined()
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

describe("fortificationOf", () => {
    it("reads the province, the flag and the tile that made it whole", () => {
        const event = new PlanetEvent({
            event: {case: "provinceFortified", value: new ProvinceFortified({provinceId: 42, countryId: "us", tileId: 1234})},
        })

        expect(fortificationOf(event)).toEqual({province: 42, countryId: "us", tile: 1234, version: NO_VERSION})
    })

    it("drops everything that is not a fortified province", () => {
        expect(fortificationOf(new PlanetEvent({event: {case: "heartbeat", value: new Heartbeat()}}))).toBeUndefined()
    })
})

describe("knockOutOf", () => {
    it("reads the flag out, the flag that took its capital, and the tiles it cleared at their version", () => {
        const event = new PlanetEvent({
            event: {case: "knockOut", value: new KnockOutMessage({
                countryId: "fr", byCountryId: "de", tileId: 4, clearedTileIds: [5, 6],
                version: new MapVersionMessage({epoch: 1_790_000_000, seq: BigInt(9)}),
            })},
        })

        expect(knockOutOf(event)).toEqual({
            countryId: "fr", byCountryId: "de", tile: 4, cleared: [5, 6], version: {epoch: 1_790_000_000, seq: 9},
        })
    })

    it("drops everything that is not a knock-out", () => {
        expect(knockOutOf(new PlanetEvent({event: {case: "heartbeat", value: new Heartbeat()}}))).toBeUndefined()
    })
})

describe("cutOffOf", () => {
    it("reads the flag cut off, the flag whose take cut it, and the tiles it cleared at their version", () => {
        const event = new PlanetEvent({
            event: {case: "cutOff", value: new CutOffMessage({
                countryId: "fr", byCountryId: "de", tileId: 4, clearedTileIds: [5, 6],
                version: new MapVersionMessage({epoch: 1_790_000_000, seq: BigInt(9)}),
            })},
        })

        expect(cutOffOf(event)).toEqual({
            countryId: "fr", byCountryId: "de", tile: 4, cleared: [5, 6], version: {epoch: 1_790_000_000, seq: 9},
        })
    })

    it("drops everything that is not a cut-off", () => {
        expect(cutOffOf(new PlanetEvent({event: {case: "heartbeat", value: new Heartbeat()}}))).toBeUndefined()
    })
})

describe("bombOf", () => {
    const dropped = (fields: Partial<{tileId: number, clearedTileIds: number[], struck: TileShields[]}> = {}) => new PlanetEvent({
        event: {
            case: "bombDropped",
            value: new BombDropped({
                tileId: 7,
                countryId: "fr",
                radius: 0.03,
                clearedTileIds: [6, 7, 8],
                struck: [new TileShields({tileId: 5, shields: 2})],
                point: new GlobePoint({x: 0, y: 1, z: 0}),
                version: new MapVersionMessage({epoch: 1_790_000_000, seq: BigInt(12)}),
                ...fields,
            }),
        },
    })

    it("reads a blast on land, with every tile it cleared and the shields each tile it struck kept", () => {
        expect(bombOf(dropped())).toEqual({
            tile: 7,
            point: {x: 0, y: 1, z: 0},
            countryId: "fr",
            radius: 0.03,
            cleared: [6, 7, 8],
            struck: [{tile: 5, shields: 2}],
            version: {epoch: 1_790_000_000, seq: 12},
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

function versionAt(seq: number): MapVersionMessage {
    return new MapVersionMessage({epoch: 1_790_000_000, seq: BigInt(seq)})
}

function heartbeatAt(seq: number): PlanetEvent {
    return new PlanetEvent({event: {case: "heartbeat", value: new Heartbeat({version: versionAt(seq)})}})
}

describe("countsOf", () => {
    it("reads a server from before the versions as version 0", () => {
        const counts: Counts = countsOf(new GetHoldingsResponse())
        const none: MapVersion = {epoch: 0, seq: 0}

        expect(counts.version).toEqual(none)
    })
})

describe("PlanetBackend bombs", () => {
    const clientWith = (fields: Record<string, unknown>) =>
        ({click: vi.fn(), getMap: vi.fn(), getBudget: noBudget(), ...bonusReads(), mapDensity: vi.fn(), listenForEvents: noEvents(), ...fields}) as never

    it("sends where the player aimed and the session token", async () => {
        const dropBomb = vi.fn().mockResolvedValue({})
        const backend = new PlanetBackend(clientWith({dropBomb}), 1_000, fixedSession("session-1"))

        await backend.dropBomb({x: 1, y: 2, z: 3}, "fr")

        expect(dropBomb).toHaveBeenCalledWith({target: {x: 1, y: 2, z: 3}, countryId: "fr"}, expect.anything())
        expect((dropBomb.mock.calls[0][1] as {headers: Headers}).headers.get(SESSION_HEADER)).toBe("session-1")
        backend.close()
    })

    it("reports a bomb refused for a ban as a BannedError", async () => {
        const dropBomb = vi.fn().mockRejectedValue(new ConnectError("the caller is banned", Code.PermissionDenied, undefined,
            [new BanRefusal({bannedUntilUnixMs: BigInt(1_700_003_600_000)})]))
        const backend = new PlanetBackend(clientWith({dropBomb}), 1_000)

        await expect(backend.dropBomb({x: 1, y: 0, z: 0}, "fr")).rejects.toBeInstanceOf(BannedError)
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

    it("hands over the take of a capital before the knock-out it made", async () => {
        const events = [
            tileUpdateEvent({tileId: 4, countryId: "de"}),
            new PlanetEvent({
                event: {case: "knockOut", value: new KnockOutMessage({countryId: "fr", byCountryId: "de", tileId: 4, clearedTileIds: [5]})},
            }),
        ]
        const listenForEvents = vi.fn(async function* () {
            yield* events
            await new Promise(() => {})
        })
        const backend = new PlanetBackend(clientWith({listenForEvents}), 60_000)

        const order: string[] = []
        backend.listenForUpdatesBatch(() => order.push("updates"))
        backend.listenForKnockOuts(() => order.push("knock-out"))

        await vi.waitFor(() => expect(order).toEqual(["updates", "knock-out"]))
        backend.close()
    })

    it("hands over the take that cut a flag's land before the cut-off it made", async () => {
        const events = [
            tileUpdateEvent({tileId: 4, countryId: "de"}),
            new PlanetEvent({
                event: {case: "cutOff", value: new CutOffMessage({countryId: "fr", byCountryId: "de", tileId: 4, clearedTileIds: [5]})},
            }),
        ]
        const listenForEvents = vi.fn(async function* () {
            yield* events
            await new Promise(() => {})
        })
        const backend = new PlanetBackend(clientWith({listenForEvents}), 60_000)

        const order: string[] = []
        backend.listenForUpdatesBatch(() => order.push("updates"))
        backend.listenForCutOffs(() => order.push("cut-off"))

        await vi.waitFor(() => expect(order).toEqual(["updates", "cut-off"]))
        backend.close()
    })

    it("hands over the tile updates that came before an enclosure, before the enclosure", async () => {
        const events = [
            tileUpdateEvent({tileId: 7, countryId: "jp"}),
            new PlanetEvent({
                event: {case: "tilesEnclosed", value: new TilesEnclosed({countryId: "fr", closingTileId: 6, filledTileIds: [7]})},
            }),
        ]
        const listenForEvents = vi.fn(async function* () {
            yield* events
            await new Promise(() => {})
        })
        const backend = new PlanetBackend(clientWith({listenForEvents}), 60_000)

        const order: string[] = []
        backend.listenForUpdatesBatch(() => order.push("updates"))
        backend.listenForBonuses({
            onVirusSpread: () => {}, onMines: () => {}, onMineLaid: () => {}, onTraps: () => {}, onTrapLaid: () => {}, onTrapSprung: () => {}, onMinesExploded: () => {},
            onRallies: () => {},
            onRallyCalled: () => {}, onEnclosed: () => order.push("enclosure"),
            onSpread: () => {}, onCharges: () => {}, onRules: () => {},
        })

        await vi.waitFor(() => expect(order).toEqual(["updates", "enclosure"]))
        backend.close()
    })

    it("hands over the tile updates that came before a gap, before saying the stream opened again", async () => {
        let connections = 0
        const listenForEvents = vi.fn(async function* () {
            connections++
            yield heartbeatAt(connections * 10)
            yield tileUpdateEvent({tileId: connections, countryId: "jp", version: versionAt(connections * 10 + 1)})
            if (connections > 1) await new Promise(() => {})
        })
        const backend = new PlanetBackend(clientWith({listenForEvents}), 60_000)

        const order: string[] = []
        backend.listenForUpdatesBatch((updates) => order.push(`updates ${updates.map(({tile}) => tile).join(",")}`))
        backend.listenForOpenings(({seq}) => order.push(`opened ${seq}`))

        await vi.waitFor(() => expect(order).toEqual(["opened 10", "updates 1", "opened 20"]), {timeout: 2_000})
        backend.close()
    })

    it("tells a listener that comes late the version the stream already passed, its updates handed over first", async () => {
        const listenForEvents = vi.fn(async function* () {
            yield heartbeatAt(10)
            yield tileUpdateEvent({tileId: 3, countryId: "jp", version: versionAt(11)})
            await new Promise(() => {})
        })
        const backend = new PlanetBackend(clientWith({listenForEvents}), 60_000)
        const order: string[] = []
        backend.listenForUpdatesBatch((updates) => order.push(`updates ${updates.map(({tile}) => tile).join(",")}`))
        const heard: number[] = []
        backend.listenForUpdates(({tile}) => heard.push(tile))
        await vi.waitFor(() => expect(heard).toEqual([3]))

        backend.listenForOpenings(({seq}) => order.push(`opened ${seq}`))

        expect(order).toEqual(["updates 3", "opened 11"])
        backend.close()
    })

    it("hands over the tile updates before the counts they moved, and maps the counts", async () => {
        const listenForEvents = vi.fn(async function* () {
            yield heartbeatAt(10)
            yield tileUpdateEvent({tileId: 3, countryId: "jp", version: versionAt(11)})
            yield new PlanetEvent({event: {case: "holdingsChanged", value: new HoldingsChanged({
                version: versionAt(11),
                countries: [new Holding({countryId: "jp", tiles: 4}), new Holding({countryId: "fr"})],
                landmasses: [new LandmassHoldings({landmassId: 7, holdings: [new Holding({countryId: "jp", tiles: 4})]})],
                provinces: [new ProvinceHoldings({provinceId: 31, holdings: [new Holding({countryId: "jp", tiles: 1})]})],
            })}})
            await new Promise(() => {})
        })
        const backend = new PlanetBackend(clientWith({listenForEvents}), 60_000)

        const order: string[] = []
        const changes: Counts[] = []
        backend.listenForUpdatesBatch(() => order.push("updates"))
        backend.listenForHoldings((change) => {
            order.push("holdings")
            changes.push(change)
        })

        await vi.waitFor(() => expect(order).toEqual(["updates", "holdings"]))
        expect(changes).toEqual([{
            version: {epoch: 1_790_000_000, seq: 11},
            countries: new Map([["jp", 4], ["fr", 0]]),
            landmasses: new Map([[7, new Map([["jp", 4]])]]),
            provinces: new Map([[31, new Map([["jp", 1]])]]),
        }])
        backend.close()
    })

    it("reads what each flag holds", async () => {
        const getHoldings = vi.fn().mockResolvedValue(new GetHoldingsResponse({
            version: versionAt(30),
            countries: [new Holding({countryId: "fr", tiles: 3})],
            landmasses: [new LandmassHoldings({landmassId: 2, holdings: [new Holding({countryId: "fr", tiles: 3})]})],
            provinces: [new ProvinceHoldings({provinceId: 5, holdings: [new Holding({countryId: "fr", tiles: 2})]})],
        }))
        const backend = new PlanetBackend(clientWith({getHoldings}), 60_000)

        expect(await backend.getHoldings()).toEqual({
            version: {epoch: 1_790_000_000, seq: 30},
            countries: new Map([["fr", 3]]),
            landmasses: new Map([[2, new Map([["fr", 3]])]]),
            provinces: new Map([[5, new Map([["fr", 2]])]]),
        })
        backend.close()
    })
})

describe("PlanetBackend shields", () => {
    const clientWith = (fields: Record<string, unknown>) =>
        ({click: vi.fn(), getMap: vi.fn(), getBudget: noBudget(), ...bonusReads(), mapDensity: vi.fn(), listenForEvents: noEvents(), ...fields}) as never

    const follow = (backend: PlanetBackend) => {
        const seen: Charges[] = []
        backend.listenForBonuses({
            onVirusSpread: () => {}, onMines: () => {}, onMineLaid: () => {}, onTraps: () => {}, onTrapLaid: () => {}, onTrapSprung: () => {}, onMinesExploded: () => {},
            onRallies: () => {},
            onRallyCalled: () => {}, onEnclosed: () => {}, onSpread: () => {}, onRules: () => {},
            onCharges: (charges) => seen.push(charges),
        })
        return seen
    }

    it("places a shield with the session token, and holds the charges the server answers", async () => {
        const getCharges = vi.fn().mockResolvedValue({charges: new ChargesHeld({shields: 5})})
        const placeShield = vi.fn().mockResolvedValue({charges: new ChargesHeld({shields: 4})})
        const backend = new PlanetBackend(clientWith({getCharges, placeShield}), 1_000, fixedSession("session-1"))
        const seen = follow(backend)
        await vi.waitFor(() => expect(seen.at(-1)?.shields).toBe(5))

        await backend.placeShield(7, "fr")

        expect(placeShield).toHaveBeenCalledWith({tileId: 7, countryId: "fr"}, expect.anything())
        expect((placeShield.mock.calls[0][1] as {headers: Headers}).headers.get(SESSION_HEADER)).toBe("session-1")
        expect(seen.at(-1)?.shields).toBe(4)
        backend.close()
    })

    it("takes the shield off at once, and gives it back when the server refused the tile", async () => {
        const getCharges = vi.fn().mockResolvedValue({charges: new ChargesHeld({shields: 5})})
        const placeShield = vi.fn().mockRejectedValue(new ConnectError("full", Code.FailedPrecondition))
        const backend = new PlanetBackend(clientWith({getCharges, placeShield}), 1_000)
        const seen = follow(backend)
        await vi.waitFor(() => expect(seen.at(-1)?.shields).toBe(5))

        const placing = backend.placeShield(7, "fr")
        expect(seen.at(-1)?.shields).toBe(4)
        await expect(placing).rejects.toBeInstanceOf(ShieldRefusedError)

        expect(seen.at(-1)?.shields).toBe(5)
        backend.close()
    })

    it("gives the shield back when the call never reached the server", async () => {
        const getCharges = vi.fn().mockResolvedValue({charges: new ChargesHeld({shields: 2})})
        const placeShield = vi.fn().mockRejectedValue(new ConnectError("down", Code.Internal))
        const backend = new PlanetBackend(clientWith({getCharges, placeShield}), 1_000)
        const seen = follow(backend)
        await vi.waitFor(() => expect(seen.at(-1)?.shields).toBe(2))

        await expect(backend.placeShield(7, "fr")).rejects.toBeDefined()

        expect(seen.at(-1)?.shields).toBe(2)
        backend.close()
    })

    it("holds none when the server had no shield to place", async () => {
        const getCharges = vi.fn().mockResolvedValue({charges: new ChargesHeld({shields: 2})})
        const placeShield = vi.fn().mockRejectedValue(new ConnectError("none", Code.NotFound))
        const backend = new PlanetBackend(clientWith({getCharges, placeShield}), 1_000)
        const seen = follow(backend)
        await vi.waitFor(() => expect(seen.at(-1)?.shields).toBe(2))

        await expect(backend.placeShield(7, "fr")).rejects.toBeInstanceOf(BonusLostError)

        expect(seen.at(-1)?.shields).toBe(0)
        backend.close()
    })

    it("mints a new session and places once more when the server refuses the token", async () => {
        const placeShield = vi.fn()
            .mockRejectedValueOnce(new ConnectError("expired", Code.Unauthenticated))
            .mockResolvedValueOnce({charges: new ChargesHeld()})
        const session = rotatingSession(["stale", "fresh"])
        const backend = new PlanetBackend(clientWith({placeShield}), 1_000, session)

        await backend.placeShield(7, "fr")

        expect(session.invalidated).toBe(1)
        expect(placeShield.mock.calls.map(([, options]) => (options as {headers: Headers}).headers.get(SESSION_HEADER)))
            .toEqual(["stale", "fresh"])
        backend.close()
    })
})

describe("PlanetBackend viruses", () => {
    const clientWith = (fields: Record<string, unknown>) =>
        ({click: vi.fn(), getMap: vi.fn(), getBudget: noBudget(), ...bonusReads(), mapDensity: vi.fn(), listenForEvents: noEvents(), ...fields}) as never

    const follow = (backend: PlanetBackend) => {
        const seen: Charges[] = []
        backend.listenForBonuses({
            onVirusSpread: () => {}, onMines: () => {}, onMineLaid: () => {}, onTraps: () => {}, onTrapLaid: () => {}, onTrapSprung: () => {}, onMinesExploded: () => {},
            onRallies: () => {},
            onRallyCalled: () => {}, onEnclosed: () => {}, onSpread: () => {}, onRules: () => {},
            onCharges: (charges) => seen.push(charges),
        })
        return seen
    }

    it("uses the virus on the tile with the session token, and takes it off the hand", async () => {
        const getCharges = vi.fn().mockResolvedValue({charges: new ChargesHeld({viruses: 1})})
        const useVirus = vi.fn().mockResolvedValue({})
        const backend = new PlanetBackend(clientWith({getCharges, useVirus}), 1_000, fixedSession("session-1"))
        const seen = follow(backend)
        await vi.waitFor(() => expect(seen.at(-1)?.viruses).toBe(1))

        await backend.useVirus(7, "fr")

        expect(useVirus).toHaveBeenCalledWith({tileId: 7, countryId: "fr"}, expect.anything())
        expect((useVirus.mock.calls[0][1] as {headers: Headers}).headers.get(SESSION_HEADER)).toBe("session-1")
        expect(seen.at(-1)?.viruses).toBe(0)
        backend.close()
    })

    it("gives the virus back when the server refused the tile", async () => {
        const getCharges = vi.fn().mockResolvedValue({charges: new ChargesHeld({viruses: 1})})
        const refused = new ConnectError("shielded", Code.FailedPrecondition, undefined, [new VirusRefusal()])
        const useVirus = vi.fn().mockRejectedValue(refused)
        const backend = new PlanetBackend(clientWith({getCharges, useVirus}), 1_000)
        const seen = follow(backend)
        await vi.waitFor(() => expect(seen.at(-1)?.viruses).toBe(1))

        const using = backend.useVirus(7, "fr")
        expect(seen.at(-1)?.viruses).toBe(0)
        await expect(using).rejects.toBeInstanceOf(VirusRefusedError)

        expect(seen.at(-1)?.viruses).toBe(1)
        backend.close()
    })

    it("takes one virus of several off the hand, and gives it back when the server refused the tile", async () => {
        const getCharges = vi.fn().mockResolvedValue({charges: new ChargesHeld({viruses: 2})})
        const refused = new ConnectError("shielded", Code.FailedPrecondition, undefined, [new VirusRefusal()])
        const useVirus = vi.fn().mockResolvedValueOnce({}).mockRejectedValueOnce(refused)
        const backend = new PlanetBackend(clientWith({getCharges, useVirus}), 1_000, fixedSession("session-1"))
        const seen = follow(backend)
        await vi.waitFor(() => expect(seen.at(-1)?.viruses).toBe(2))

        await backend.useVirus(7, "fr")
        expect(seen.at(-1)?.viruses).toBe(1)

        await expect(backend.useVirus(8, "fr")).rejects.toBeInstanceOf(VirusRefusedError)
        expect(seen.at(-1)?.viruses).toBe(1)
        backend.close()
    })

    it("tells a tile out of reach, and keeps the virus off when the server holds none", async () => {
        const getCharges = vi.fn().mockResolvedValue({charges: new ChargesHeld({viruses: 1})})
        const outOfReach = new ConnectError("far", Code.FailedPrecondition, undefined, [new ReachRefusal()])
        const useVirus = vi.fn().mockRejectedValueOnce(outOfReach).mockRejectedValueOnce(new ConnectError("none", Code.NotFound))
        const backend = new PlanetBackend(clientWith({getCharges, useVirus}), 1_000)
        const seen = follow(backend)
        await vi.waitFor(() => expect(seen.at(-1)?.viruses).toBe(1))

        await expect(backend.useVirus(7, "fr")).rejects.toBeInstanceOf(OutOfReachError)
        expect(seen.at(-1)?.viruses).toBe(1)
        await expect(backend.useVirus(7, "fr")).rejects.toBeInstanceOf(BonusLostError)
        expect(seen.at(-1)?.viruses).toBe(0)
        backend.close()
    })

    it("reads the tiles a live virus infects with the map", async () => {
        const getMap = vi.fn().mockResolvedValue(new GetMapResponse({
            startTileId: 1, codes: ["", "fr"], tiles: new Uint8Array([1, 0, 1, 0]),
            infected: [new InfectedTile({tileId: 2, virusId: "v-1"})], version: versionAt(5),
        }))
        const backend = new PlanetBackend(clientWith({getMap}), 2, fixedSession("session-1"))
        const read: Map<number, string>[] = []

        await backend.getOwnerships([{start: 1, end: 2}], (ownerships) => read.push(ownerships.infected))

        expect(read).toEqual([new Map([[2, "v-1"]])])
        backend.close()
    })
})

describe("PlanetBackend event stream session", () => {
    const openForever = () => vi.fn<(req: object, options: {signal: AbortSignal, headers: Headers}) => AsyncIterable<PlanetEvent>>(
        () => (async function* () {
            await new Promise(() => {})
            yield* []
        })(),
    )

    const clientWith = (fields: Record<string, unknown>) =>
        ({click: vi.fn().mockResolvedValue({}), getMap: vi.fn(), getBudget: noBudget(), ...bonusReads(), mapDensity: vi.fn(), ...fields}) as never

    const tokensOpenedWith = (listenForEvents: ReturnType<typeof openForever>) =>
        listenForEvents.mock.calls.map(call => call[1].headers.get(SESSION_HEADER))

    it("opens the stream with the token in hand, so the server knows whose account it serves", async () => {
        const listenForEvents = openForever()
        const backend = new PlanetBackend(clientWith({listenForEvents}), 1_000, fixedSession("session-1"))

        await vi.waitFor(() => expect(tokensOpenedWith(listenForEvents)).toEqual(["session-1"]))
        backend.close()
    })

    it("follows the identity the cookie resumes, reads the budget and charges with it, and never mints", async () => {
        const listenForEvents = openForever()
        const getBudget = vi.fn().mockResolvedValue({})
        const getCharges = vi.fn().mockResolvedValue({charges: new ChargesHeld()})
        const token = vi.fn(async () => "minted")
        let resumed: string | undefined
        const session: SessionProvider = {
            token,
            held: () => undefined,
            identity: async () => resumed = "identity-1",
            heldIdentity: () => resumed,
            invalidate: () => {},
        }
        const backend = new PlanetBackend(clientWith({listenForEvents, getBudget, getCharges}), 1_000, session)

        await vi.waitFor(() => expect(tokensOpenedWith(listenForEvents).at(-1)).toBe("identity-1"))
        await vi.waitFor(() => expect((getCharges.mock.calls.at(-1)?.[1].headers as Headers).get(SESSION_HEADER)).toBe("identity-1"))
        expect((getBudget.mock.calls[0][1].headers as Headers).get(SESSION_HEADER)).toBe("identity-1")
        expect(token).not.toHaveBeenCalled()
        backend.close()
    })

    it("opens it with no token when none is held, and never mints one for it", async () => {
        const listenForEvents = openForever()
        const token = vi.fn(async () => "minted")
        const session: SessionProvider = {token, held: () => undefined, identity: async () => undefined, heldIdentity: () => undefined, invalidate: () => {}}
        const backend = new PlanetBackend(clientWith({listenForEvents}), 1_000, session)

        await vi.waitFor(() => expect(tokensOpenedWith(listenForEvents)).toEqual([null]))
        expect(token).not.toHaveBeenCalled()
        backend.close()
    })

    it("reopens the stream once a click goes out under a token the stream does not have", async () => {
        const listenForEvents = openForever()
        let held: string | undefined
        const session: SessionProvider = {
            token: async () => {
                held = "session-1"
                return held
            },
            held: () => held, identity: async () => held, heldIdentity: () => held,
            invalidate: () => {
                held = undefined
            },
        }
        const backend = new PlanetBackend(clientWith({listenForEvents}), 1_000, session)
        await vi.waitFor(() => expect(tokensOpenedWith(listenForEvents)).toEqual([null]))
        const first = listenForEvents.mock.calls[0][1].signal

        await backend.clickTile(1, "fr")

        await vi.waitFor(() => expect(tokensOpenedWith(listenForEvents)).toEqual([null, "session-1"]))
        expect(first.aborted).toBe(true)

        await backend.clickTile(2, "fr")
        expect(tokensOpenedWith(listenForEvents)).toEqual([null, "session-1"])
        backend.close()
    })

    it("keeps the stream when the click was refused", async () => {
        vi.spyOn(console, "error").mockImplementation(() => {})
        const listenForEvents = openForever()
        const click = vi.fn().mockRejectedValue(new ConnectError("slow down", Code.ResourceExhausted))
        let held: string | undefined
        const session: SessionProvider = {
            token: async () => {
                held = "session-1"
                return held
            },
            held: () => held, identity: async () => held, heldIdentity: () => held,
            invalidate: () => {},
        }
        const backend = new PlanetBackend(clientWith({listenForEvents, click}), 1_000, session)
        await vi.waitFor(() => expect(tokensOpenedWith(listenForEvents)).toEqual([null]))

        await expect(backend.clickTile(1, "fr")).rejects.toBeInstanceOf(RateLimitedError)

        expect(tokensOpenedWith(listenForEvents)).toEqual([null])
        backend.close()
    })
})

describe("PlanetBackend sessions", () => {
    const clientWith = (fields: Record<string, unknown> = {}) =>
        ({click: vi.fn().mockResolvedValue({}), getMap: vi.fn(), getBudget: noBudget(), ...bonusReads(), mapDensity: vi.fn(),
            listenForEvents: noEvents(), ...fields}) as never

    it("tells the session listeners each token a call carried, so the boxes' and the quizzes' streams follow it", async () => {
        const backend = new PlanetBackend(clientWith(), 1_000, fixedSession("session-1"), bonusesWith())
        const carried: string[] = []
        backend.listenForSessions((token) => carried.push(token))

        await backend.clickTile(3, "fr")

        expect(carried).toContain("session-1")
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

    it("reads a ban on a bomb or a shield, and leaves any other denial alone", () => {
        const banned = new ConnectError("the caller is banned", Code.PermissionDenied, undefined,
            [new BanRefusal({bannedUntilUnixMs: BigInt(1_700_003_600_000)})])
        expect(asBonusError(banned)).toMatchObject({until: 1_700_003_600_000})

        const denied = new ConnectError("no", Code.PermissionDenied)
        expect(asBonusError(denied)).toBe(denied)
    })

    it("leaves anything else alone", () => {
        const boom = new Error("boom")
        expect(asBonusError(boom)).toBe(boom)
    })
})

describe("the rules", () => {
    const clientWith = (fields: Record<string, unknown>) =>
        ({click: vi.fn(), getMap: vi.fn(), getBudget: noBudget(), ...bonusReads(), mapDensity: vi.fn(),
            listenForEvents: noEvents(), ...fields}) as never

    const follow = (backend: PlanetBackend) => {
        const seen: BonusRules[] = []
        backend.listenForBonuses({
            onVirusSpread: () => {}, onMines: () => {}, onMineLaid: () => {}, onTraps: () => {}, onTrapLaid: () => {}, onTrapSprung: () => {}, onMinesExploded: () => {},
            onRallies: () => {},
            onRallyCalled: () => {}, onEnclosed: () => {}, onSpread: () => {}, onCharges: () => {},
            onRules: (rules) => seen.push(rules),
        })
        return seen
    }

    it("reads the toll, the sizes of the charges, the sign-in refill, unlimited enclosures and spreads, the reach and the capitals' rules", async () => {
        const getChargeRules = vi.fn().mockResolvedValue(new GetChargeRulesResponse({
            blastRadius: 0.03, enclosureMaxTiles: 25, spreadClicks: 8, enclosures: 3, shields: 12, tileShields: 10,
            tollSteps: [{share: 0.1, slowdown: 1.5}, {share: 0.3, slowdown: 4}], enclosuresUnlimited: true,
            reach: true, knockOut: true, capitalShieldsFixed: false, bombsOff: true, spreadUnlimited: true,
            virusInfections: 32, refills: 1, bombs: 2, viruses: 3,
            mines: 3, mineRings: 2, rallies: 3, rallyRadius: 0.072, rallyLastsMs: 180_000, rallyCost: 0.5,
            traps: 3, trapClicks: 10,
        }))
        const getGrantRules = vi.fn().mockResolvedValue(new GetGrantRulesResponse({signInRefill: true}))
        const backend = new PlanetBackend(clientWith({getChargeRules}), 1_000, new NoSession(), bonusesWith({getGrantRules}))

        const seen = follow(backend)

        await vi.waitFor(() => expect(seen.at(-1)).toEqual({
            blastRadius: 0.03, enclosureMaxTiles: 25, spreadClicks: 8, enclosures: 3, shields: 12, tileShields: 10,
            toll: [{share: 0.1, slowdown: 1.5}, {share: 0.3, slowdown: 4}], signInRefill: true, enclosuresUnlimited: true,
            reach: true, knockOut: true, capitalShieldsFixed: false, bombsOff: true, spreadUnlimited: true,
            enclosuresOff: false, spreadOff: false, ralliesOff: false, virusInfections: 32, refills: 1, bombs: 2, viruses: 3, gatesOn: false, openKinds: [],
            mines: 3, mineRings: 2, rallies: 3, rallyRadius: 0.072, rallyLastsMs: 180_000, rallyCost: 0.5,
            traps: 3, trapClicks: 10,
        }))
        backend.close()
    })

    it("reads enclosures off", async () => {
        const getChargeRules = vi.fn().mockResolvedValue(new GetChargeRulesResponse({enclosuresOff: true}))
        const backend = new PlanetBackend(clientWith({getChargeRules}), 1_000)
        let off = false
        backend.listenForBonuses({
            onVirusSpread: () => {}, onMines: () => {}, onMineLaid: () => {}, onTraps: () => {}, onTrapLaid: () => {}, onTrapSprung: () => {}, onMinesExploded: () => {},
            onRallies: () => {},
            onRallyCalled: () => {}, onEnclosed: () => {}, onSpread: () => {}, onCharges: () => {},
            onRules: (rules) => off = rules.enclosuresOff,
        })

        await vi.waitFor(() => expect(off).toBe(true))
        backend.close()
    })

    it("reads spreads off", async () => {
        const getChargeRules = vi.fn().mockResolvedValue(new GetChargeRulesResponse({spreadOff: true}))
        const backend = new PlanetBackend(clientWith({getChargeRules}), 1_000)
        let off = false
        backend.listenForBonuses({
            onVirusSpread: () => {}, onMines: () => {}, onMineLaid: () => {}, onTraps: () => {}, onTrapLaid: () => {}, onTrapSprung: () => {}, onMinesExploded: () => {},
            onRallies: () => {},
            onRallyCalled: () => {}, onEnclosed: () => {}, onSpread: () => {}, onCharges: () => {},
            onRules: (rules) => off = rules.spreadOff,
        })

        await vi.waitFor(() => expect(off).toBe(true))
        backend.close()
    })

    it("reads rallies off", async () => {
        const getChargeRules = vi.fn().mockResolvedValue(new GetChargeRulesResponse({ralliesOff: true}))
        const backend = new PlanetBackend(clientWith({getChargeRules}), 1_000)
        let off = false
        backend.listenForBonuses({
            onVirusSpread: () => {}, onMines: () => {}, onMineLaid: () => {}, onTraps: () => {}, onTrapLaid: () => {}, onTrapSprung: () => {}, onMinesExploded: () => {},
            onRallies: () => {},
            onRallyCalled: () => {}, onEnclosed: () => {}, onSpread: () => {}, onCharges: () => {},
            onRules: (rules) => off = rules.ralliesOff,
        })

        await vi.waitFor(() => expect(off).toBe(true))
        backend.close()
    })

    it("tells the sign-in refill the bonuses' rules say with planet's rules, whichever is read first", async () => {
        let chargeRulesRead: (res: GetChargeRulesResponse) => void = () => {}
        const getChargeRules = vi.fn(() => new Promise<GetChargeRulesResponse>(resolve => chargeRulesRead = resolve))
        const getGrantRules = vi.fn().mockResolvedValue(new GetGrantRulesResponse({signInRefill: true}))
        const backend = new PlanetBackend(clientWith({getChargeRules}), 1_000, new NoSession(), bonusesWith({getGrantRules}))
        const seen = follow(backend)
        await vi.waitFor(() => expect(getGrantRules).toHaveBeenCalledOnce())
        await new Promise(resolve => setTimeout(resolve, 0))

        expect(seen).toEqual([])
        chargeRulesRead(new GetChargeRulesResponse({enclosureMaxTiles: 25}))

        await vi.waitFor(() => expect(seen.map(({signInRefill}) => signInRefill)).toEqual([true]))
        backend.close()
    })

    it("tells planet's rules again once the bonuses' rules say a first sign-in earns a refill", async () => {
        let grantRulesRead: (res: GetGrantRulesResponse) => void = () => {}
        const getGrantRules = vi.fn(() => new Promise<GetGrantRulesResponse>(resolve => grantRulesRead = resolve))
        const backend = new PlanetBackend(clientWith({}), 1_000, new NoSession(), bonusesWith({getGrantRules}))
        const seen = follow(backend)
        await vi.waitFor(() => expect(seen.map(({signInRefill}) => signInRefill)).toEqual([false]))

        grantRulesRead(new GetGrantRulesResponse({signInRefill: true}))

        await vi.waitFor(() => expect(seen.map(({signInRefill}) => signInRefill)).toEqual([false, true]))
        expect(seen[1]).toEqual({...seen[0], signInRefill: true})
        backend.close()
    })

    it("tells the gates the bonuses' rules say, read again on a refresh, and nothing when they did not change", async () => {
        const getGrantRules = vi.fn()
            .mockResolvedValueOnce(new GetGrantRulesResponse())
            .mockResolvedValue(new GetGrantRulesResponse({
                gatesOn: true, openKinds: [BonusKind.REFILL, BonusKind.SHIELDS, BonusKind.UNSPECIFIED],
            }))
        const backend = new PlanetBackend(clientWith({}), 1_000, new NoSession(), bonusesWith({getGrantRules}))
        const seen = follow(backend)
        await vi.waitFor(() => expect(seen.at(-1)).toMatchObject({gatesOn: false, openKinds: []}))
        await vi.waitFor(() => expect(getGrantRules).toHaveBeenCalledOnce())

        await backend.refreshGrantRules()
        expect(seen.at(-1)).toMatchObject({gatesOn: true, openKinds: ["refill", "shields"]})

        const told = seen.length
        await backend.refreshGrantRules()
        expect(seen).toHaveLength(told)
        backend.close()
    })

    it("leaves the bonuses' rules be when planet's are read again: the turn reads those", async () => {
        const getChargeRules = vi.fn().mockResolvedValue(new GetChargeRulesResponse({enclosureMaxTiles: 25}))
        const getGrantRules = vi.fn().mockResolvedValue(new GetGrantRulesResponse({signInRefill: true}))
        let changeRules = () => {}
        const listenForEvents = vi.fn(async function* () {
            yield new PlanetEvent({event: {case: "heartbeat", value: new Heartbeat()}})
            await new Promise<void>(resolve => changeRules = resolve)
            yield new PlanetEvent({event: {case: "rulesChanged", value: new RulesChanged()}})
            await new Promise(() => {})
        })
        const backend = new PlanetBackend(clientWith({getChargeRules, listenForEvents}), 1_000, new NoSession(),
            bonusesWith({getGrantRules}))
        const seen = follow(backend)
        await vi.waitFor(() => expect(seen.at(-1)?.signInRefill).toBe(true))

        changeRules()

        await vi.waitFor(() => expect(getChargeRules).toHaveBeenCalledTimes(2))
        expect(getGrantRules).toHaveBeenCalledOnce()
        expect(seen.at(-1)?.signInRefill).toBe(true)
        backend.close()
    })
    it("reads them again when the rules change, so a bigger blast is drawn with no reload", async () => {
        const getChargeRules = vi.fn()
            .mockResolvedValueOnce(new GetChargeRulesResponse({blastRadius: 0.03, enclosureMaxTiles: 25}))
            .mockResolvedValueOnce(new GetChargeRulesResponse({blastRadius: 0.05, enclosureMaxTiles: 40}))
        let changeRules = () => {}
        const listenForEvents = vi.fn(async function* () {
            yield new PlanetEvent({event: {case: "heartbeat", value: new Heartbeat()}})
            await new Promise<void>(resolve => changeRules = resolve)
            yield new PlanetEvent({event: {case: "rulesChanged", value: new RulesChanged()}})
            await new Promise(() => {})
        })
        const backend = new PlanetBackend(clientWith({getChargeRules, listenForEvents}), 1_000)
        const radii: number[] = []
        backend.listenForBonuses({
            onVirusSpread: () => {}, onMines: () => {}, onMineLaid: () => {}, onTraps: () => {}, onTrapLaid: () => {}, onTrapSprung: () => {}, onMinesExploded: () => {},
            onRallies: () => {},
            onRallyCalled: () => {}, onEnclosed: () => {}, onSpread: () => {}, onCharges: () => {},
            onRules: (rules) => radii.push(rules.blastRadius),
        })
        await vi.waitFor(() => expect(radii.at(-1)).toBe(0.03))

        changeRules()

        await vi.waitFor(() => expect(radii.at(-1)).toBe(0.05))
        backend.close()
    })
})

describe("the charges held", () => {
    const clientWith = (fields: Record<string, unknown>) =>
        ({click: vi.fn().mockResolvedValue({}), getMap: vi.fn(), getBudget: noBudget(), ...bonusReads(), mapDensity: vi.fn(),
            listenForEvents: noEvents(), ...fields}) as never

    const follow = (backend: PlanetBackend) => {
        const seen: Charges[] = []
        backend.listenForBonuses({
            onVirusSpread: () => {}, onMines: () => {}, onMineLaid: () => {}, onTraps: () => {}, onTrapLaid: () => {}, onTrapSprung: () => {}, onMinesExploded: () => {},
            onRallies: () => {},
            onRallyCalled: () => {}, onEnclosed: () => {}, onSpread: () => {}, onRules: () => {},
            onCharges: (charges) => seen.push(charges),
        })
        return seen
    }

    it("reads them once at load, and hands them to whoever listens", async () => {
        const getCharges = vi.fn().mockResolvedValue({charges: new ChargesHeld({bombs: 1, spreadClicksLeft: 2, shields: 5})})
        const backend = new PlanetBackend(clientWith({getCharges}), 1_000)

        const seen = follow(backend)
        await vi.waitFor(() => expect(seen.at(-1)).toEqual({refills: 0, bombs: 1, enclosures: 0, spreadClicksLeft: 2, shields: 5, viruses: 0, mines: 0, rallies: 0, traps: 0}))

        expect(getCharges).toHaveBeenCalledTimes(1)
        backend.close()
    })

    it("holds what a grant on the stream says, with no read", async () => {
        const getCharges = vi.fn().mockResolvedValue({charges: new ChargesHeld({shields: 5})})
        let grant = () => {}
        const listenForEvents = vi.fn(async function* () {
            yield new PlanetEvent({event: {case: "heartbeat", value: new Heartbeat()}})
            await new Promise<void>(resolve => grant = resolve)
            yield new PlanetEvent({event: {case: "chargesGranted", value: new ChargesGranted({charges: new ChargesHeld({bombs: 1, shields: 25})})}})
            await new Promise(() => {})
        })
        const backend = new PlanetBackend(clientWith({getCharges, listenForEvents}), 1_000)
        const seen = follow(backend)
        await vi.waitFor(() => expect(seen.at(-1)?.shields).toBe(5))

        grant()

        await vi.waitFor(() => expect(seen.at(-1)).toEqual({refills: 0, bombs: 1, enclosures: 0, spreadClicksLeft: 0, shields: 25, viruses: 0, mines: 0, rallies: 0, traps: 0}))
        expect(getCharges).toHaveBeenCalledTimes(1)
        backend.close()
    })

    it("tells the grant listeners what a grant added, each kind as a reward", async () => {
        const listenForEvents = vi.fn(async function* () {
            yield new PlanetEvent({event: {case: "heartbeat", value: new Heartbeat()}})
            yield new PlanetEvent({event: {case: "chargesGranted", value: new ChargesGranted({
                charges: new ChargesHeld({bombs: 1, shields: 25}),
                gained: [
                    new ChargeGained({kind: BonusKind.SHIELDS, amount: 20}),
                    new ChargeGained({kind: BonusKind.UNSPECIFIED, amount: 1}),
                ],
            })}})
            yield new PlanetEvent({event: {case: "chargesGranted", value: new ChargesGranted({charges: new ChargesHeld({shields: 25})})}})
            await new Promise(() => {})
        })
        const backend = new PlanetBackend(clientWith({listenForEvents}), 1_000)
        const granted: BonusReward[][] = []
        backend.listenForGrants((rewards) => granted.push(rewards))

        await vi.waitFor(() => expect(granted).toEqual([[{kind: "shields", shields: 20}]]))
        backend.close()
    })

    it("tells the grant listeners a kind paid into a full slot, with none kept", async () => {
        const listenForEvents = vi.fn(async function* () {
            yield new PlanetEvent({event: {case: "heartbeat", value: new Heartbeat()}})
            yield new PlanetEvent({event: {case: "chargesGranted", value: new ChargesGranted({
                charges: new ChargesHeld({spreadClicksLeft: 8}),
                gained: [new ChargeGained({kind: BonusKind.SPREAD_CLICKS, amount: 0})],
            })}})
            await new Promise(() => {})
        })
        const backend = new PlanetBackend(clientWith({listenForEvents}), 1_000)
        const granted: BonusReward[][] = []
        backend.listenForGrants((rewards) => granted.push(rewards))

        await vi.waitFor(() => expect(granted).toEqual([[{kind: "spreadClicks", clicks: 0}]]))
        backend.close()
    })

    it("counts a spread click off each accepted click sent with spread on, since the click answers nothing about it", async () => {
        const getCharges = vi.fn().mockResolvedValue({charges: new ChargesHeld({spreadClicksLeft: 2})})
        const click = vi.fn()
            .mockResolvedValueOnce({})
            .mockResolvedValueOnce({})
            .mockRejectedValueOnce(new ConnectError("slow down", Code.ResourceExhausted))
        const backend = new PlanetBackend(clientWith({getCharges, click}), 1_000)
        const seen = follow(backend)
        await vi.waitFor(() => expect(seen.at(-1)?.spreadClicksLeft).toBe(2))

        await backend.clickTile(1, "fr")
        expect(seen.at(-1)?.spreadClicksLeft).toBe(2)

        await backend.clickTile(2, "fr", {spread: true, enclose: false, shield: false})
        await expect(backend.clickTile(3, "fr", {spread: true, enclose: false, shield: false})).rejects.toBeInstanceOf(RateLimitedError)

        expect(seen.at(-1)?.spreadClicksLeft).toBe(1)
        backend.close()
    })

    it("counts no spread click off while spreads are unlimited", async () => {
        const getCharges = vi.fn().mockResolvedValue({charges: new ChargesHeld({spreadClicksLeft: 2})})
        const getChargeRules = vi.fn().mockResolvedValue(new GetChargeRulesResponse({spreadUnlimited: true}))
        const click = vi.fn().mockResolvedValue({})
        const backend = new PlanetBackend(clientWith({getCharges, getChargeRules, click}), 1_000)
        const seen = follow(backend)
        let unlimited = false
        backend.listenForBonuses({
            onVirusSpread: () => {}, onMines: () => {}, onMineLaid: () => {}, onTraps: () => {}, onTrapLaid: () => {}, onTrapSprung: () => {}, onMinesExploded: () => {},
            onRallies: () => {},
            onRallyCalled: () => {}, onEnclosed: () => {}, onSpread: () => {}, onCharges: () => {},
            onRules: (rules) => unlimited = rules.spreadUnlimited,
        })
        await vi.waitFor(() => expect(seen.at(-1)?.spreadClicksLeft).toBe(2))
        await vi.waitFor(() => expect(unlimited).toBe(true))

        await backend.clickTile(2, "fr", {spread: true, enclose: false, shield: false})

        expect(seen.at(-1)?.spreadClicksLeft).toBe(2)
        backend.close()
    })

    it("counts no spread click off while spreads are off", async () => {
        const getCharges = vi.fn().mockResolvedValue({charges: new ChargesHeld({spreadClicksLeft: 2})})
        const getChargeRules = vi.fn().mockResolvedValue(new GetChargeRulesResponse({spreadOff: true}))
        const click = vi.fn().mockResolvedValue({})
        const backend = new PlanetBackend(clientWith({getCharges, getChargeRules, click}), 1_000)
        const seen = follow(backend)
        let off = false
        backend.listenForBonuses({
            onVirusSpread: () => {}, onMines: () => {}, onMineLaid: () => {}, onTraps: () => {}, onTrapLaid: () => {}, onTrapSprung: () => {}, onMinesExploded: () => {},
            onRallies: () => {},
            onRallyCalled: () => {}, onEnclosed: () => {}, onSpread: () => {}, onCharges: () => {},
            onRules: (rules) => off = rules.spreadOff,
        })
        await vi.waitFor(() => expect(seen.at(-1)?.spreadClicksLeft).toBe(2))
        await vi.waitFor(() => expect(off).toBe(true))

        await backend.clickTile(2, "fr", {spread: true, enclose: false, shield: false})

        expect(seen.at(-1)?.spreadClicksLeft).toBe(2)
        backend.close()
    })

    it("sends the switches with the click", async () => {
        const click = vi.fn().mockResolvedValue({})
        const backend = new PlanetBackend(clientWith({click}), 1_000)

        await backend.clickTile(7, "fr", {spread: true, enclose: true, shield: false})
        await backend.clickTile(8, "fr")

        expect(click.mock.calls.map(([req]) => req)).toEqual([
            {tileId: 7, countryId: "fr", spread: true, enclose: true},
            {tileId: 8, countryId: "fr", spread: false, enclose: false},
        ])
        backend.close()
    })

    it("takes an enclosure off on this player's own closed shape, and not on anybody else's", async () => {
        const events: PlanetEvent[] = []
        let push: () => void = () => {}
        const listenForEvents = vi.fn(async function* () {
            while (true) {
                const next = events.shift()
                if (next) {
                    yield next
                    continue
                }
                await new Promise<void>(resolve => push = resolve)
            }
        })
        const getCharges = vi.fn().mockResolvedValue({charges: new ChargesHeld({enclosures: 2})})
        const backend = new PlanetBackend(clientWith({getCharges, listenForEvents}), 1_000)
        const seen = follow(backend)
        await vi.waitFor(() => expect(seen.at(-1)?.enclosures).toBe(2))

        const shape = (yours: boolean) => new PlanetEvent({
            event: {case: "tilesEnclosed", value: new TilesEnclosed({countryId: "fr", closingTileId: 4, yours})},
        })
        events.push(shape(false))
        push()
        await vi.waitFor(() => expect(events).toHaveLength(0))
        expect(seen.at(-1)?.enclosures).toBe(2)

        events.push(shape(true))
        push()
        await vi.waitFor(() => expect(seen.at(-1)?.enclosures).toBe(1))
        backend.close()
    })

    it("takes the bomb off at once on a drop, and gives it back when the drop never reached the server", async () => {
        const getCharges = vi.fn().mockResolvedValue({charges: new ChargesHeld({bombs: 1})})
        const dropBomb = vi.fn().mockRejectedValue(new ConnectError("down", Code.Internal))
        const backend = new PlanetBackend(clientWith({getCharges, dropBomb}), 1_000)
        const seen = follow(backend)
        await vi.waitFor(() => expect(seen.at(-1)?.bombs).toBe(1))

        const dropping = backend.dropBomb({x: 0, y: 0, z: 1}, "fr")
        expect(seen.at(-1)?.bombs).toBe(0)
        await expect(dropping).rejects.toBeDefined()

        expect(seen.at(-1)?.bombs).toBe(1)
        backend.close()
    })

    it("takes one bomb of several off on a drop, gives that one back on a refusal, and holds none when the server has none", async () => {
        const getCharges = vi.fn().mockResolvedValue({charges: new ChargesHeld({bombs: 2})})
        const dropBomb = vi.fn()
            .mockResolvedValueOnce({})
            .mockRejectedValueOnce(new ConnectError("bombs are off", Code.FailedPrecondition, undefined, [new BombRefusal()]))
            .mockRejectedValueOnce(new ConnectError("no bomb", Code.NotFound))
        const backend = new PlanetBackend(clientWith({getCharges, dropBomb}), 1_000)
        const seen = follow(backend)
        await vi.waitFor(() => expect(seen.at(-1)?.bombs).toBe(2))

        await backend.dropBomb({x: 0, y: 0, z: 1}, "fr")
        expect(seen.at(-1)?.bombs).toBe(1)

        const refused = backend.dropBomb({x: 0, y: 0, z: 1}, "fr")
        expect(seen.at(-1)?.bombs).toBe(0)
        await expect(refused).rejects.toBeInstanceOf(BombRefusedError)
        expect(seen.at(-1)?.bombs).toBe(1)

        await expect(backend.dropBomb({x: 0, y: 0, z: 1}, "fr")).rejects.toBeInstanceOf(BonusLostError)
        expect(seen.at(-1)?.bombs).toBe(0)
        backend.close()
    })

    it("reads them again when a click goes out under a new token: they are the account's", async () => {
        const getCharges = vi.fn().mockResolvedValue({charges: new ChargesHeld()})
        let held: string | undefined
        const session: SessionProvider = {
            token: async () => held = "session-1",
            held: () => held, identity: async () => held, heldIdentity: () => held,
            invalidate: () => {},
        }
        const backend = new PlanetBackend(clientWith({getCharges}), 1_000, session)
        await vi.waitFor(() => expect(getCharges).toHaveBeenCalledTimes(1))
        expect((getCharges.mock.calls[0][1].headers as Headers).has(SESSION_HEADER)).toBe(false)

        await backend.clickTile(1, "fr")

        await vi.waitFor(() => expect(getCharges).toHaveBeenCalledTimes(2))
        const headers = getCharges.mock.calls[1][1].headers as Headers
        expect(headers.get(SESSION_HEADER)).toBe("session-1")
        backend.close()
    })
})

describe("the refill", () => {
    const clientWith = (fields: Record<string, unknown>) =>
        ({click: vi.fn().mockResolvedValue({}), getMap: vi.fn(), getBudget: noBudget(), ...bonusReads(), mapDensity: vi.fn(),
            listenForEvents: noEvents(), ...fields}) as never

    const follow = (backend: PlanetBackend) => {
        const seen: Charges[] = []
        backend.listenForBonuses({
            onVirusSpread: () => {}, onMines: () => {}, onMineLaid: () => {}, onTraps: () => {}, onTrapLaid: () => {}, onTrapSprung: () => {}, onMinesExploded: () => {},
            onRallies: () => {},
            onRallyCalled: () => {}, onEnclosed: () => {}, onSpread: () => {}, onRules: () => {},
            onCharges: (charges) => seen.push(charges),
        })
        return seen
    }

    it("fills the meter from the answer and takes the refill off the charges", async () => {
        const getCharges = vi.fn().mockResolvedValue({charges: new ChargesHeld({refills: 1})})
        const useRefill = vi.fn().mockResolvedValue({budget: new ClickBudgetMessage({tokens: 60, capacity: 60, refillPerSecond: 0.2}), charges: new ChargesHeld()})
        const backend = new PlanetBackend(clientWith({getCharges, useRefill}), 1_000)
        const seen = follow(backend)
        await vi.waitFor(() => expect(seen.at(-1)?.refills).toBe(1))
        const tokens: number[] = []
        backend.watchClickBudget(b => tokens.push(b.tokens))

        await backend.useRefill("fr")

        expect(useRefill).toHaveBeenCalledWith({countryId: "fr"}, expect.anything())
        expect(tokens.at(-1)).toBe(60)
        expect(seen.at(-1)?.refills).toBe(0)
        backend.close()
    })

    it("reads a full bank as BankFullError and keeps the refill", async () => {
        const getCharges = vi.fn().mockResolvedValue({charges: new ChargesHeld({refills: 1})})
        const useRefill = vi.fn().mockRejectedValue(new ConnectError("full", Code.FailedPrecondition))
        const backend = new PlanetBackend(clientWith({getCharges, useRefill}), 1_000)
        const seen = follow(backend)
        await vi.waitFor(() => expect(seen.at(-1)?.refills).toBe(1))

        await expect(backend.useRefill("fr")).rejects.toBeInstanceOf(BankFullError)

        expect(seen.at(-1)?.refills).toBe(1)
        backend.close()
    })

    it("reads no refill as BonusLostError and takes it off the charges", async () => {
        const getCharges = vi.fn().mockResolvedValue({charges: new ChargesHeld({refills: 1})})
        const useRefill = vi.fn().mockRejectedValue(new ConnectError("none", Code.NotFound))
        const backend = new PlanetBackend(clientWith({getCharges, useRefill}), 1_000)
        const seen = follow(backend)
        await vi.waitFor(() => expect(seen.at(-1)?.refills).toBe(1))

        await expect(backend.useRefill("fr")).rejects.toBeInstanceOf(BonusLostError)

        expect(seen.at(-1)?.refills).toBe(0)
        backend.close()
    })
})

describe("PlanetBackend allegiance", () => {
    const clientWith = (fields: Record<string, unknown>) =>
        ({click: vi.fn(), getMap: vi.fn(), getBudget: noBudget(), ...bonusReads(), mapDensity: vi.fn(), listenForEvents: noEvents(), ...fields}) as never

    const held = (countryId: string, changeableAt?: number) => new GetAllegianceResponse({
        allegiance: new AllegianceMessage({countryId, changeableAtUnixMs: BigInt(changeableAt ?? 0)}),
    })

    const lockedTo = (countryId: string, changeableAt: number) => new ConnectError("too soon", Code.FailedPrecondition, undefined,
        [new AllegianceMessage({countryId, changeableAtUnixMs: BigInt(changeableAt)})])

    const follow = (backend: PlanetBackend) => {
        const seen: (Allegiance | undefined)[] = []
        backend.listenForAllegiance((allegiance) => seen.push(allegiance))
        return seen
    }

    it("reads the flag played for at load, as the identity the player holds", async () => {
        const getAllegiance = vi.fn().mockResolvedValue(held("fr"))
        const backend = new PlanetBackend(clientWith({getAllegiance}), 1_000, fixedSession("identity-1"))
        const seen = follow(backend)

        await vi.waitFor(() => expect(seen.at(-1)).toEqual({countryId: "fr"}))
        expect((getAllegiance.mock.calls[0][1] as {headers: Headers}).headers.get(SESSION_HEADER)).toBe("identity-1")
        backend.close()
    })

    it("reads the flag again under the token the session renews, so a kept token the server refuses costs no click", async () => {
        // A token kept from another network is read as nobody's, who holds no flag.
        const getAllegiance = vi.fn(async (_req: object, {headers}: {headers: Headers}) =>
            headers.get(SESSION_HEADER) === "identity-2" ? held("fr") : new GetAllegianceResponse())
        const session = renewingSession("identity-1")
        const backend = new PlanetBackend(clientWith({getAllegiance}), 1_000, session)
        const seen = follow(backend)
        await vi.waitFor(() => expect(getAllegiance).toHaveBeenCalledOnce())

        session.renew("identity-2")

        await vi.waitFor(() => expect(seen.at(-1)).toEqual({countryId: "fr"}))
        backend.close()
    })

    it("follows no token once closed", async () => {
        const getAllegiance = vi.fn().mockResolvedValue(held("fr"))
        const session = renewingSession("identity-1")
        const backend = new PlanetBackend(clientWith({getAllegiance}), 1_000, session)
        await vi.waitFor(() => expect(getAllegiance).toHaveBeenCalledOnce())
        backend.close()

        session.renew("identity-2")

        expect(getAllegiance).toHaveBeenCalledOnce()
    })

    it("keeps the flag a pledge gave when a read sent before it answers late", async () => {
        let land: (res: GetAllegianceResponse) => void = () => {}
        const getAllegiance = vi.fn()
            .mockImplementationOnce(() => new Promise(resolve => land = resolve))
            .mockResolvedValueOnce(new GetAllegianceResponse())
            .mockResolvedValue(held("de"))
        const pledgeAllegiance = vi.fn().mockResolvedValue(new PledgeAllegianceResponse())
        const click = vi.fn().mockResolvedValue({})
        const backend = new PlanetBackend(clientWith({getAllegiance, pledgeAllegiance, click}), 1_000, fixedSession("session-1"))
        const seen = follow(backend)
        await vi.waitFor(() => expect(getAllegiance).toHaveBeenCalledOnce())
        await backend.clickTile(7, "de")
        expect(seen.at(-1)).toEqual({countryId: "de"})

        land(new GetAllegianceResponse())
        await new Promise(resolve => setTimeout(resolve, 0))

        expect(seen.at(-1)).toEqual({countryId: "de"})
        backend.close()
    })

    it("reads the flag again when the rules change, so a lock shows with no reload", async () => {
        const getAllegiance = vi.fn()
            .mockResolvedValueOnce(held("fr"))
            .mockResolvedValueOnce(new GetAllegianceResponse({allegiance: new AllegianceMessage({countryId: "fr", locked: true})}))
        let changeRules = () => {}
        const listenForEvents = vi.fn(async function* () {
            yield new PlanetEvent({event: {case: "heartbeat", value: new Heartbeat()}})
            await new Promise<void>(resolve => changeRules = resolve)
            yield new PlanetEvent({event: {case: "rulesChanged", value: new RulesChanged()}})
            await new Promise(() => {})
        })
        const backend = new PlanetBackend(clientWith({getAllegiance, listenForEvents}), 1_000, fixedSession("identity-1"))
        const seen = follow(backend)
        await vi.waitFor(() => expect(seen.at(-1)).toEqual({countryId: "fr"}))

        changeRules()

        await vi.waitFor(() => expect(seen.at(-1)).toEqual({countryId: "fr", locked: true}))
        backend.close()
    })

    it("pledges the flag clicked for before the first click of an account that holds none, and only then", async () => {
        const calls: string[] = []
        const getAllegiance = vi.fn()
            .mockImplementationOnce(async () => new GetAllegianceResponse())
            .mockImplementationOnce(async () => new GetAllegianceResponse())
            .mockImplementation(async () => held("de"))
        const pledgeAllegiance = vi.fn().mockImplementation(async () => {
            calls.push("pledge")
            return new PledgeAllegianceResponse()
        })
        const click = vi.fn().mockImplementation(async () => {
            calls.push("click")
            return {}
        })
        const session = sessionMinting("session-1")
        const backend = new PlanetBackend(clientWith({getAllegiance, pledgeAllegiance, click}), 1_000, session)
        const seen = follow(backend)

        await Promise.all([backend.clickTile(7, "de"), backend.clickTile(8, "de")])
        await backend.clickTile(9, "de")

        expect(calls).toEqual(["pledge", "click", "click", "click"])
        expect(pledgeAllegiance.mock.calls[0][0]).toEqual({countryId: "de"})
        expect((pledgeAllegiance.mock.calls[0][1] as {headers: Headers}).headers.get(SESSION_HEADER)).toBe("session-1")
        expect(seen.at(-1)).toEqual({countryId: "de"})
        backend.close()
    })

    it("pledges nothing before the click of an account that holds a flag", async () => {
        const getAllegiance = vi.fn().mockResolvedValue(held("fr"))
        const pledgeAllegiance = vi.fn()
        const click = vi.fn().mockResolvedValue({})
        const backend = new PlanetBackend(clientWith({getAllegiance, pledgeAllegiance, click}), 1_000, fixedSession("session-1"))
        const seen = follow(backend)
        await vi.waitFor(() => expect(seen.at(-1)).toEqual({countryId: "fr"}))

        await backend.clickTile(7, "fr")
        await backend.clickTile(8, "fr")

        expect(pledgeAllegiance).not.toHaveBeenCalled()
        expect(getAllegiance).toHaveBeenCalledTimes(1)
        expect(click).toHaveBeenCalledTimes(2)
        backend.close()
    })

    it("claims the flag the clicks are priced for, for an account that holds none, and the first click pledges no more", async () => {
        const getAllegiance = vi.fn().mockResolvedValue(new GetAllegianceResponse())
        const pledgeAllegiance = vi.fn().mockResolvedValue(new PledgeAllegianceResponse())
        const click = vi.fn().mockResolvedValue({})
        const backend = new PlanetBackend(clientWith({getAllegiance, pledgeAllegiance, click}), 1_000, sessionMinting("session-1"))
        backend.priceFor("bg")

        await backend.claimFlag()
        expect(pledgeAllegiance).toHaveBeenCalledTimes(1)
        expect(pledgeAllegiance.mock.calls[0][0]).toEqual({countryId: "bg"})
        expect((pledgeAllegiance.mock.calls[0][1] as {headers: Headers}).headers.get(SESSION_HEADER)).toBe("session-1")

        await backend.clickTile(7, "bg")
        expect(pledgeAllegiance).toHaveBeenCalledTimes(1)
        expect(click).toHaveBeenCalledTimes(1)
        backend.close()
    })

    it("claims nothing for an account that holds a flag", async () => {
        const getAllegiance = vi.fn().mockResolvedValue(held("fr"))
        const pledgeAllegiance = vi.fn()
        const backend = new PlanetBackend(clientWith({getAllegiance, pledgeAllegiance}), 1_000, fixedSession("session-1"))
        backend.priceFor("fr")

        await backend.claimFlag()

        expect(pledgeAllegiance).not.toHaveBeenCalled()
        backend.close()
    })

    it("claims nothing before the page names its flag", async () => {
        const getAllegiance = vi.fn().mockResolvedValue(new GetAllegianceResponse())
        const pledgeAllegiance = vi.fn()
        const backend = new PlanetBackend(clientWith({getAllegiance, pledgeAllegiance}), 1_000, sessionMinting("session-1"))

        await backend.claimFlag()

        expect(pledgeAllegiance).not.toHaveBeenCalled()
        backend.close()
    })

    it("sends no click when the pledge before it failed, and tries again on the next click", async () => {
        const getAllegiance = vi.fn().mockResolvedValue(new GetAllegianceResponse())
        const pledgeAllegiance = vi.fn()
            .mockRejectedValueOnce(new ConnectError("bad flag", Code.InvalidArgument))
            .mockResolvedValue(new PledgeAllegianceResponse())
        const click = vi.fn().mockResolvedValue({})
        const backend = new PlanetBackend(clientWith({getAllegiance, pledgeAllegiance, click}), 1_000, sessionMinting("session-1"))

        await expect(backend.clickTile(7, "fr")).rejects.toBeDefined()
        expect(click).not.toHaveBeenCalled()

        await backend.clickTile(7, "fr")
        expect(pledgeAllegiance).toHaveBeenCalledTimes(2)
        expect(click).toHaveBeenCalledTimes(1)
        backend.close()
    })

    it("turns a click refused for its flag into the flag held and until when", async () => {
        const click = vi.fn().mockRejectedValue(lockedTo("de", 1_700_000_000_000))
        const backend = new PlanetBackend(clientWith({click}), 1_000, fixedSession("session-1"))
        const seen = follow(backend)

        const clicking = backend.clickTile(7, "fr")

        await expect(clicking).rejects.toBeInstanceOf(AllegianceLockedError)
        await expect(clicking).rejects.toMatchObject({allegiance: {countryId: "de", changeableAt: 1_700_000_000_000}})
        expect(seen.at(-1)).toEqual({countryId: "de", changeableAt: 1_700_000_000_000})
        backend.close()
    })

    it("turns a shield refused for its flag into the flag held, and gives the shield back", async () => {
        const getCharges = vi.fn().mockResolvedValue({charges: new ChargesHeld({shields: 3})})
        const placeShield = vi.fn().mockRejectedValue(lockedTo("de", 1_700_000_000_000))
        const backend = new PlanetBackend(clientWith({getCharges, placeShield}), 1_000, fixedSession("session-1"))
        const seen = follow(backend)
        const charges: Charges[] = []
        backend.listenForBonuses({
            onVirusSpread: () => {}, onMines: () => {}, onMineLaid: () => {}, onTraps: () => {}, onTrapLaid: () => {}, onTrapSprung: () => {}, onMinesExploded: () => {},
            onRallies: () => {},
            onRallyCalled: () => {}, onEnclosed: () => {}, onSpread: () => {}, onRules: () => {},
            onCharges: (held) => charges.push(held),
        })
        await vi.waitFor(() => expect(charges.at(-1)?.shields).toBe(3))

        await expect(backend.placeShield(7, "fr")).rejects.toBeInstanceOf(AllegianceLockedError)

        expect(seen.at(-1)).toEqual({countryId: "de", changeableAt: 1_700_000_000_000})
        expect(charges.at(-1)?.shields).toBe(3)
        backend.close()
    })

    it("pledges a flag as the identity the player holds, then holds the flag the server keeps", async () => {
        const getAllegiance = vi.fn()
            .mockResolvedValueOnce(held("fr"))
            .mockResolvedValue(held("de", 1_700_000_000_000))
        const pledgeAllegiance = vi.fn().mockResolvedValue(new PledgeAllegianceResponse())
        const backend = new PlanetBackend(clientWith({getAllegiance, pledgeAllegiance}), 1_000, fixedSession("identity-1"))
        const seen = follow(backend)
        await vi.waitFor(() => expect(seen.at(-1)).toEqual({countryId: "fr"}))

        await backend.pledgeAllegiance("de")

        expect(pledgeAllegiance.mock.calls[0][0]).toEqual({countryId: "de"})
        expect((pledgeAllegiance.mock.calls[0][1] as {headers: Headers}).headers.get(SESSION_HEADER)).toBe("identity-1")
        expect(seen.at(-1)).toEqual({countryId: "de", changeableAt: 1_700_000_000_000})
        backend.close()
    })

    it("pledges nothing for a visitor with no account: its first click will", async () => {
        const pledgeAllegiance = vi.fn()
        const backend = new PlanetBackend(clientWith({pledgeAllegiance}), 1_000, sessionMinting("session-1"))

        await backend.pledgeAllegiance("de")

        expect(pledgeAllegiance).not.toHaveBeenCalled()
        backend.close()
    })

    it("turns a pledge refused into the flag held and until when", async () => {
        const pledgeAllegiance = vi.fn().mockRejectedValue(lockedTo("fr", 1_700_000_000_000))
        const backend = new PlanetBackend(clientWith({pledgeAllegiance}), 1_000, fixedSession("identity-1"))
        const seen = follow(backend)

        await expect(backend.pledgeAllegiance("de")).rejects.toBeInstanceOf(AllegianceLockedError)

        expect(seen.at(-1)).toEqual({countryId: "fr", changeableAt: 1_700_000_000_000})
        backend.close()
    })

    it("sends the click to a server that takes no pledge, as before", async () => {
        const getAllegiance = vi.fn().mockResolvedValue(new GetAllegianceResponse())
        const pledgeAllegiance = vi.fn().mockRejectedValue(new ConnectError("no such procedure", Code.Unimplemented))
        const click = vi.fn().mockResolvedValue({})
        const backend = new PlanetBackend(clientWith({getAllegiance, pledgeAllegiance, click}), 1_000, sessionMinting("session-1"))

        await backend.clickTile(7, "de")

        expect(click).toHaveBeenCalledTimes(1)
        backend.close()
    })

    it("holds no flag from a server that does not know one", async () => {
        const getAllegiance = vi.fn().mockRejectedValue(new ConnectError("no such procedure", Code.Unimplemented))
        const backend = new PlanetBackend(clientWith({getAllegiance}), 1_000, fixedSession("identity-1"))
        const seen = follow(backend)

        await vi.waitFor(() => expect(getAllegiance).toHaveBeenCalled())
        expect(seen.at(-1)).toBeUndefined()
        backend.close()
    })
})

describe("PlanetBackend tile holder", () => {
    const backendWith = (getTileHolder: ReturnType<typeof vi.fn>) => new PlanetBackend(
        {click: vi.fn(), getMap: vi.fn(), getBudget: noBudget(), ...bonusReads(), mapDensity: vi.fn(), listenForEvents: noEvents(), getTileHolder} as never,
        1_000,
    )

    it("reads who holds a tile, with no token, and when it was taken", async () => {
        const getTileHolder = vi.fn().mockResolvedValue(new GetTileHolderResponse({
            holder: new TileHolderMessage({accountId: "account-ana", countryId: "fr", takenAtUnixMs: 1_761_393_600_000n}),
        }))
        const backend = backendWith(getTileHolder)

        expect(await backend.tileHolder(7)).toEqual({accountId: "account-ana", countryCode: "fr", takenAt: 1_761_393_600_000})
        expect(getTileHolder.mock.calls[0][0]).toEqual({tileId: 7})
        expect((getTileHolder.mock.calls[0][1] as {headers?: Headers}).headers).toBeUndefined()
        backend.close()
    })

    it("answers nobody for a tile the ledger names nobody on", async () => {
        const backend = backendWith(vi.fn().mockResolvedValue(new GetTileHolderResponse()))

        expect(await backend.tileHolder(7)).toBeUndefined()
        backend.close()
    })
})

describe("PlanetBackend landmass", () => {
    const backendWith = (getLandmass: ReturnType<typeof vi.fn>) => new PlanetBackend(
        {click: vi.fn(), getMap: vi.fn(), getBudget: noBudget(), ...bonusReads(), mapDensity: vi.fn(), listenForEvents: noEvents(), getLandmass} as never,
        1_000,
    )

    it("reads who holds a landmass, with no token", async () => {
        const getLandmass = vi.fn().mockResolvedValue(new GetLandmassResponse({
            holders: [
                new LandmassHolderMessage({accountId: "account-ana", countryId: "fr", tiles: 40}),
                new LandmassHolderMessage({accountId: "account-kiran", countryId: "in", tiles: 3}),
            ],
        }))
        const backend = backendWith(getLandmass)

        expect(await backend.landmass(12)).toEqual({
            holders: [
                {accountId: "account-ana", countryCode: "fr", tiles: 40},
                {accountId: "account-kiran", countryCode: "in", tiles: 3},
            ],
        })
        expect(getLandmass.mock.calls[0][0]).toEqual({landmassId: 12})
        expect((getLandmass.mock.calls[0][1] as {headers?: Headers}).headers).toBeUndefined()
        backend.close()
    })
})

describe("PlanetBackend province", () => {
    const backendWith = (getProvince: ReturnType<typeof vi.fn>) => new PlanetBackend(
        {click: vi.fn(), getMap: vi.fn(), getBudget: noBudget(), ...bonusReads(), mapDensity: vi.fn(), listenForEvents: noEvents(), getProvince} as never,
        1_000,
    )

    it("reads who fortified a province last, with no token", async () => {
        const getProvince = vi.fn().mockResolvedValue(new GetProvinceResponse({
            lastFortification: new ProvinceFortificationMessage({accountId: "account-ana", countryId: "fr", fortifiedAtUnixMs: 1_761_393_600_000n}),
        }))
        const backend = backendWith(getProvince)

        expect(await backend.province(1031)).toEqual({
            lastFortification: {accountId: "account-ana", countryCode: "fr", at: 1_761_393_600_000},
        })
        expect(getProvince.mock.calls[0][0]).toEqual({provinceId: 1031})
        expect((getProvince.mock.calls[0][1] as {headers?: Headers}).headers).toBeUndefined()
        backend.close()
    })

    it("reads each lock by its province", async () => {
        const getFortresses = vi.fn().mockResolvedValue(new GetFortressesResponse({
            fortresses: [new FortressMessage({provinceId: 1031, countryId: "fr"}), new FortressMessage({provinceId: 7, countryId: "de"})],
        }))
        const backend = new PlanetBackend(
            {click: vi.fn(), getMap: vi.fn(), getBudget: noBudget(), ...bonusReads(), mapDensity: vi.fn(), listenForEvents: noEvents(), getFortresses} as never,
            1_000,
        )

        expect(await backend.getFortresses()).toEqual(new Map([[1031, "fr"], [7, "de"]]))
        backend.close()
    })

    it("answers no fortify for a province nobody fortified", async () => {
        const backend = backendWith(vi.fn().mockResolvedValue(new GetProvinceResponse()))

        expect(await backend.province(1031)).toEqual({lastFortification: undefined})
        backend.close()
    })
})

describe("the mines' events", () => {
    it("reads each blast of an explosion in order, and its version", () => {
        const event = new PlanetEvent({event: {case: "minesExploded", value: new MinesExplodedMessage({
            blasts: [
                new MineBlastMessage({tileId: 5, countryId: "fr", takenTileIds: [5, 4], struck: [{tileId: 6, shields: 2}]}),
                new MineBlastMessage({tileId: 9, countryId: "de"}),
            ],
            version: versionAt(44),
        })}})

        expect(minesExplodedOf(event)).toEqual({
            blasts: [
                {tile: 5, countryId: "fr", taken: [5, 4], struck: [{tile: 6, shields: 2}]},
                {tile: 9, countryId: "de", taken: [], struck: []},
            ],
            version: {epoch: versionAt(44).epoch, seq: 44},
        })
        expect(minesExplodedOf(new PlanetEvent({event: {case: "heartbeat", value: new Heartbeat()}}))).toBeUndefined()
    })

    it("reads a mine laid with the version it was laid at", () => {
        const event = new PlanetEvent({event: {case: "mineLaid", value: new MineLaidMessage({
            countryId: "fr", tileId: 9, version: versionAt(3),
        })}})

        expect(mineLaidOf(event)).toEqual({countryId: "fr", tile: 9, version: {epoch: versionAt(3).epoch, seq: 3}})
    })
})

describe("PlanetBackend mines", () => {
    const clientWith = (fields: Record<string, unknown>) =>
        ({click: vi.fn(), getMap: vi.fn(), getBudget: noBudget(), ...bonusReads(), mapDensity: vi.fn(), listenForEvents: noEvents(), ...fields}) as never

    const follow = (backend: PlanetBackend) => {
        const seen: Charges[] = []
        const fields: Minefield[] = []
        backend.listenForBonuses({
            onVirusSpread: () => {}, onMineLaid: () => {}, onTraps: () => {}, onTrapLaid: () => {}, onTrapSprung: () => {}, onMinesExploded: () => {},
            onRallies: () => {},
            onRallyCalled: () => {}, onEnclosed: () => {}, onSpread: () => {},
            onRules: () => {}, onCharges: (charges) => seen.push(charges), onMines: (mines) => fields.push(mines),
        })
        return {seen, fields}
    }

    it("lays a mine on the tile with the session token, and takes one off the hand", async () => {
        const getCharges = vi.fn().mockResolvedValue({charges: new ChargesHeld({mines: 3})})
        const layMine = vi.fn().mockResolvedValue({})
        const backend = new PlanetBackend(clientWith({getCharges, layMine}), 1_000, fixedSession("session-1"))
        const {seen} = follow(backend)
        await vi.waitFor(() => expect(seen.at(-1)?.mines).toBe(3))

        await backend.layMine(7, "fr")

        expect(layMine).toHaveBeenCalledWith({tileId: 7, countryId: "fr"}, expect.anything())
        expect((layMine.mock.calls[0][1] as {headers: Headers}).headers.get(SESSION_HEADER)).toBe("session-1")
        expect(seen.at(-1)?.mines).toBe(2)
        backend.close()
    })

    it("gives the mine back when the server refused the tile, and holds none when the server says so", async () => {
        const getCharges = vi.fn().mockResolvedValue({charges: new ChargesHeld({mines: 2})})
        const refused = new ConnectError("not yours", Code.FailedPrecondition, undefined, [new MineRefusal()])
        const layMine = vi.fn().mockRejectedValueOnce(refused).mockRejectedValueOnce(new ConnectError("none", Code.NotFound))
        const backend = new PlanetBackend(clientWith({getCharges, layMine}), 1_000)
        const {seen} = follow(backend)
        await vi.waitFor(() => expect(seen.at(-1)?.mines).toBe(2))

        await expect(backend.layMine(7, "fr")).rejects.toBeInstanceOf(MineRefusedError)
        expect(seen.at(-1)?.mines).toBe(2)
        await expect(backend.layMine(7, "fr")).rejects.toBeInstanceOf(BonusLostError)
        expect(seen.at(-1)?.mines).toBe(0)
        backend.close()
    })

    it("reads the flag's mines as the stream opens", async () => {
        const getMines = vi.fn().mockResolvedValue(new GetMinesResponse({countryId: "fr", tileIds: [3, 8], version: versionAt(10)}))
        const listenForEvents = vi.fn(async function* () {
            yield heartbeatAt(10)
            await new Promise(() => {})
        })
        const backend = new PlanetBackend(clientWith({getMines, listenForEvents}), 60_000, fixedSession("session-1"))
        const {fields} = follow(backend)

        await vi.waitFor(() => expect(fields.at(-1)).toEqual({
            countryId: "fr", tiles: [3, 8], version: {epoch: versionAt(10).epoch, seq: 10},
        }))
        backend.close()
    })

    it("hands over the tile updates that came before an explosion, before the explosion", async () => {
        const events = [
            tileUpdateEvent({tileId: 5, countryId: "de"}),
            new PlanetEvent({event: {case: "minesExploded", value: new MinesExplodedMessage({
                blasts: [new MineBlastMessage({tileId: 5, countryId: "fr", takenTileIds: [5]})],
            })}}),
        ]
        const listenForEvents = vi.fn(async function* () {
            yield* events
            await new Promise(() => {})
        })
        const backend = new PlanetBackend(clientWith({listenForEvents}), 60_000)

        const order: string[] = []
        backend.listenForUpdatesBatch(() => order.push("updates"))
        backend.listenForBonuses({
            onVirusSpread: () => {}, onMines: () => {}, onMineLaid: () => {}, onTraps: () => {}, onTrapLaid: () => {}, onTrapSprung: () => {}, onRallies: () => {}, onRallyCalled: () => {},
            onMinesExploded: () => order.push("explosion"),
            onEnclosed: () => {}, onSpread: () => {}, onCharges: () => {}, onRules: () => {},
        })

        await vi.waitFor(() => expect(order).toEqual(["updates", "explosion"]))
        backend.close()
    })
})

describe("PlanetBackend traps", () => {
    const clientWith = (fields: Record<string, unknown>) =>
        ({click: vi.fn(), getMap: vi.fn(), getBudget: noBudget(), ...bonusReads(), mapDensity: vi.fn(), listenForEvents: noEvents(), ...fields}) as never

    const follow = (backend: PlanetBackend) => {
        const seen: Charges[] = []
        const fields: Trapline[] = []
        const sprung: TrapSprung[] = []
        backend.listenForBonuses({
            onVirusSpread: () => {}, onMines: () => {}, onMineLaid: () => {}, onMinesExploded: () => {}, onTrapLaid: () => {},
            onRallies: () => {}, onRallyCalled: () => {}, onEnclosed: () => {}, onSpread: () => {}, onRules: () => {},
            onCharges: (charges) => seen.push(charges), onTraps: (traps) => fields.push(traps),
            onTrapSprung: (trap) => sprung.push(trap),
        })
        return {seen, fields, sprung}
    }

    it("lays a trap on the tile with the session token, and takes one off the hand", async () => {
        const getCharges = vi.fn().mockResolvedValue({charges: new ChargesHeld({traps: 3})})
        const layTrap = vi.fn().mockResolvedValue({})
        const backend = new PlanetBackend(clientWith({getCharges, layTrap}), 1_000, fixedSession("session-1"))
        const {seen} = follow(backend)
        await vi.waitFor(() => expect(seen.at(-1)?.traps).toBe(3))

        await backend.layTrap(7, "fr")

        expect(layTrap).toHaveBeenCalledWith({tileId: 7, countryId: "fr"}, expect.anything())
        expect((layTrap.mock.calls[0][1] as {headers: Headers}).headers.get(SESSION_HEADER)).toBe("session-1")
        expect(seen.at(-1)?.traps).toBe(2)
        backend.close()
    })

    it("gives the trap back when the server refused the tile, and holds none when the server says so", async () => {
        const getCharges = vi.fn().mockResolvedValue({charges: new ChargesHeld({traps: 2})})
        const refused = new ConnectError("not yours", Code.FailedPrecondition, undefined, [new TrapRefusal()])
        const layTrap = vi.fn().mockRejectedValueOnce(refused).mockRejectedValueOnce(new ConnectError("none", Code.NotFound))
        const backend = new PlanetBackend(clientWith({getCharges, layTrap}), 1_000)
        const {seen} = follow(backend)
        await vi.waitFor(() => expect(seen.at(-1)?.traps).toBe(2))

        await expect(backend.layTrap(7, "fr")).rejects.toBeInstanceOf(TrapRefusedError)
        expect(seen.at(-1)?.traps).toBe(2)
        await expect(backend.layTrap(7, "fr")).rejects.toBeInstanceOf(BonusLostError)
        expect(seen.at(-1)?.traps).toBe(0)
        backend.close()
    })

    it("reads the flag's traps as the stream opens", async () => {
        const getTraps = vi.fn().mockResolvedValue(new GetTrapsResponse({countryId: "fr", tileIds: [4], version: versionAt(10)}))
        const listenForEvents = vi.fn(async function* () {
            yield heartbeatAt(10)
            await new Promise(() => {})
        })
        const backend = new PlanetBackend(clientWith({getTraps, listenForEvents}), 60_000, fixedSession("session-1"))
        const {fields} = follow(backend)

        await vi.waitFor(() => expect(fields.at(-1)).toEqual({
            countryId: "fr", tiles: [4], version: {epoch: versionAt(10).epoch, seq: 10},
        }))
        backend.close()
    })

    it("hands over a trap sprung, and reads the bank again when the take was the page's own", async () => {
        const getBudget = noBudget()
        let yoursNext = () => {}
        const yourTurn = new Promise<void>((resolve) => yoursNext = resolve)
        const listenForEvents = vi.fn(async function* () {
            yield heartbeatAt(10)
            yield new PlanetEvent({event: {case: "trapSprung", value: new TrapSprungMessage({tileId: 5, countryId: "de", clicks: 10})}})
            await yourTurn
            yield new PlanetEvent({event: {case: "trapSprung", value: new TrapSprungMessage({
                tileId: 6, countryId: "de", clicks: 7.5, yours: true,
            })}})
            await new Promise(() => {})
        })
        const backend = new PlanetBackend(clientWith({getBudget, listenForEvents}), 60_000)
        const {sprung} = follow(backend)
        await vi.waitFor(() => expect(sprung).toHaveLength(1))
        await new Promise((settled) => setTimeout(settled, 20))
        const reads = getBudget.mock.calls.length
        yoursNext()

        await vi.waitFor(() => expect(sprung).toEqual([
            {tile: 5, countryId: "de", clicks: 10, yours: false},
            {tile: 6, countryId: "de", clicks: 7.5, yours: true},
        ]))
        await vi.waitFor(() => expect(getBudget.mock.calls.length).toBeGreaterThan(reads))
        backend.close()
    })
})

describe("PlanetBackend rallies", () => {
    const clientWith = (fields: Record<string, unknown>) =>
        ({click: vi.fn(), getMap: vi.fn(), getBudget: noBudget(), ...bonusReads(), mapDensity: vi.fn(), listenForEvents: noEvents(), ...fields}) as never

    const rallyMessage = (fields: Partial<RallyMessage> = {}) => new RallyMessage({
        rallyId: "r1", accountId: "a1", countryId: "fr", point: new GlobePoint({x: 0.6, z: 0.8}), radius: 0.072,
        leftMs: 60_000, ...fields,
    })

    const follow = (backend: PlanetBackend) => {
        const seen: Charges[] = []
        const read: Rally[][] = []
        const called: Rally[] = []
        backend.listenForBonuses({
            onVirusSpread: () => {}, onMines: () => {}, onMineLaid: () => {}, onTraps: () => {}, onTrapLaid: () => {}, onTrapSprung: () => {}, onMinesExploded: () => {}, onEnclosed: () => {},
            onSpread: () => {}, onRules: () => {}, onCharges: (charges) => seen.push(charges),
            onRallies: (rallies) => read.push(rallies), onRallyCalled: (rally) => called.push(rally),
        })
        return {seen, read, called}
    }

    it("reads a rally called with its end on the page's clock", () => {
        const event = new PlanetEvent({event: {case: "rallyCalled", value: rallyMessage()}})
        const before = performance.now()

        const rally = rallyCalledOf(event)

        expect(rally).toMatchObject({id: "r1", accountId: "a1", countryId: "fr", centre: {x: 0.6, y: 0, z: 0.8}, radius: 0.072})
        expect(rally!.endsAt).toBeGreaterThanOrEqual(before + 60_000)
        expect(rallyCalledOf(new PlanetEvent({event: {case: "heartbeat", value: new Heartbeat()}}))).toBeUndefined()
    })

    it("calls a rally with the session token, takes one off the hand, and tells its ring at once", async () => {
        const getCharges = vi.fn().mockResolvedValue({charges: new ChargesHeld({rallies: 2})})
        const callRally = vi.fn().mockResolvedValue(new CallRallyResponse({rally: rallyMessage({leftMs: 180_000})}))
        const backend = new PlanetBackend(clientWith({getCharges, callRally}), 1_000, fixedSession("session-1"))
        const {seen, called} = follow(backend)
        await vi.waitFor(() => expect(seen.at(-1)?.rallies).toBe(2))

        const rally = await backend.callRally({x: 0.6, y: 0, z: 0.8}, "fr")

        expect(callRally).toHaveBeenCalledWith({point: {x: 0.6, y: 0, z: 0.8}, countryId: "fr"}, expect.anything())
        expect((callRally.mock.calls[0][1] as {headers: Headers}).headers.get(SESSION_HEADER)).toBe("session-1")
        expect(seen.at(-1)?.rallies).toBe(1)
        expect(called).toEqual([rally])
        backend.close()
    })

    it("gives the rally back while one of the caller's is still on, and holds none when the server says so", async () => {
        const getCharges = vi.fn().mockResolvedValue({charges: new ChargesHeld({rallies: 2})})
        const refused = new ConnectError("still on", Code.FailedPrecondition, undefined, [new RallyRefusal()])
        const callRally = vi.fn().mockRejectedValueOnce(refused).mockRejectedValueOnce(new ConnectError("none", Code.NotFound))
        const backend = new PlanetBackend(clientWith({getCharges, callRally}), 1_000)
        const {seen, called} = follow(backend)
        await vi.waitFor(() => expect(seen.at(-1)?.rallies).toBe(2))

        await expect(backend.callRally({x: 1, y: 0, z: 0}, "fr")).rejects.toBeInstanceOf(RallyRefusedError)
        expect(seen.at(-1)?.rallies).toBe(2)
        await expect(backend.callRally({x: 1, y: 0, z: 0}, "fr")).rejects.toBeInstanceOf(BonusLostError)
        expect(seen.at(-1)?.rallies).toBe(0)
        expect(called).toEqual([])
        backend.close()
    })

    it("reads every rally on as the stream opens, and hears each one called on it", async () => {
        const getRallies = vi.fn().mockResolvedValue(new GetRalliesResponse({rallies: [rallyMessage({countryId: "de"})]}))
        const listenForEvents = vi.fn(async function* () {
            yield heartbeatAt(10)
            yield new PlanetEvent({event: {case: "rallyCalled", value: rallyMessage({rallyId: "r2", point: new GlobePoint({y: 1})})}})
            await new Promise(() => {})
        })
        const backend = new PlanetBackend(clientWith({getRallies, listenForEvents}), 60_000)
        const {read, called} = follow(backend)

        await vi.waitFor(() => expect(read.at(-1)?.map((rally) => rally.countryId)).toEqual(["de"]))
        await vi.waitFor(() => expect(called.map((rally) => rally.centre)).toEqual([{x: 0, y: 1, z: 0}]))
        backend.close()
    })

    it("counts a click in flight at its cost, and says the bank's clicks cost half while the flag's ring is on", async () => {
        let answer: (value: unknown) => void = () => {}
        const click = vi.fn(() => new Promise((resolve) => {
            answer = resolve
        }))
        const listenForEvents = vi.fn(async function* () {
            yield heartbeatAt(1)
            yield new PlanetEvent({event: {case: "rallyCalled", value: rallyMessage()}})
            await new Promise(() => {})
        })
        const getBudget = vi.fn().mockResolvedValue({budget: new ClickBudgetMessage({tokens: 4, capacity: 10, refillPerSecond: 0})})
        const backend = new PlanetBackend(clientWith({click, getBudget, listenForEvents}), 60_000)
        const {called} = follow(backend)
        const budgets: ClickBudget[] = []
        backend.watchClickBudget((budget) => budgets.push(budget))
        await vi.waitFor(() => expect(called).toHaveLength(1))
        await vi.waitFor(() => expect(budgets.at(-1)?.tokens).toBe(4))

        const clicked = backend.clickTile(9, "fr", ALL_OFF, 0.5)
        await vi.waitFor(() => expect(budgets.at(-1)?.tokens).toBe(3.5))
        await vi.waitFor(() => expect(click).toHaveBeenCalled())
        answer({budget: new ClickBudgetMessage({tokens: 3.5, capacity: 10, refillPerSecond: 0, spent: 0.5})})
        await clicked

        expect(budgets.at(-1)?.tokens).toBe(3.5)
        expect(budgets.at(-1)?.rally).toEqual({cost: 0.5, until: called[0].endsAt})
        backend.close()
    })
})

describe("PlanetBackend's board", () => {
    const clientWith = (fields: Record<string, unknown>) =>
        ({click: vi.fn(), getMap: vi.fn(), getBudget: noBudget(), ...bonusReads(), mapDensity: vi.fn(), listenForEvents: noEvents(), ...fields}) as never

    const changesWith = (reload: () => void, clock: {now: number}) => new BoardChanges({
        reload,
        carries: (board) => board === "earth-1" || board === "siege-1",
        memory: {read: () => undefined, write: () => {}, forget: () => {}},
        now: () => clock.now,
    })

    it("is the one GetMap names, read from its first tile alone", async () => {
        const getMap = vi.fn().mockResolvedValue(new GetMapResponse({startTileId: 1, board: "earth-1"}))
        const backend = new PlanetBackend(clientWith({getMap}), 1_000)

        await expect(backend.readBoard()).resolves.toBe("earth-1")
        expect(getMap).toHaveBeenCalledWith({startTileId: 1, endTileId: 1}, expect.anything())
        backend.close()
    })

    it("hands over no batch of another board, and reloads once a read past the browser's copy still names it", async () => {
        const reload = vi.fn()
        const clock = {now: 0}
        const boards = changesWith(reload, clock)
        await boards.choose(async () => "earth-1")
        const board = {current: "siege-1"}
        const getMap = vi.fn(async () => new GetMapResponse({startTileId: 1, codes: ["", "fr"], tiles: tileBytes([1, 1]), board: board.current}))
        const backend = new PlanetBackend(clientWith({getMap}), 1_000, undefined, undefined, boards)
        const seen: number[] = []

        await backend.getOwnerships([{start: 1, end: 2}], ({range}) => seen.push(range.start))
        expect(seen).toEqual([])
        expect(reload).not.toHaveBeenCalled()

        clock.now = CACHED_MS
        board.current = "earth-1"
        await backend.getOwnerships([{start: 1, end: 2}], ({range}) => seen.push(range.start))
        expect(seen).toEqual([1])

        board.current = "siege-1"
        await backend.getOwnerships([{start: 1, end: 2}], ({range}) => seen.push(range.start))
        clock.now = 2 * CACHED_MS
        await backend.getOwnerships([{start: 1, end: 2}], ({range}) => seen.push(range.start))
        expect(seen).toEqual([1])
        expect(reload).toHaveBeenCalledTimes(1)
        backend.close()
    })

    it("reloads when the stream opens on another board, and tells the page no opening", async () => {
        const reload = vi.fn()
        const boards = changesWith(reload, {now: 0})
        await boards.choose(async () => "earth-1")
        const listenForEvents = vi.fn(async function* () {
            yield new PlanetEvent({event: {case: "heartbeat", value: new Heartbeat({board: "siege-1", version: new MapVersionMessage({epoch: 2, seq: BigInt(0)})})}})
            await new Promise(() => {})
        })
        const backend = new PlanetBackend(clientWith({listenForEvents}), 60_000, undefined, undefined, boards)
        const openings = vi.fn()
        backend.listenForOpenings(openings)

        await vi.waitFor(() => expect(reload).toHaveBeenCalledTimes(1))
        expect(openings).not.toHaveBeenCalled()
        backend.close()
    })

    it("tells the opening of a stream on the board the page drew", async () => {
        const reload = vi.fn()
        const boards = changesWith(reload, {now: 0})
        await boards.choose(async () => "earth-1")
        const listenForEvents = vi.fn(async function* () {
            yield new PlanetEvent({event: {case: "heartbeat", value: new Heartbeat({board: "earth-1", version: new MapVersionMessage({epoch: 2, seq: BigInt(3)})})}})
            await new Promise(() => {})
        })
        const backend = new PlanetBackend(clientWith({listenForEvents}), 60_000, undefined, undefined, boards)
        const openings = vi.fn()
        backend.listenForOpenings(openings)

        await vi.waitFor(() => expect(openings).toHaveBeenCalledWith({epoch: 2, seq: 3}))
        expect(reload).not.toHaveBeenCalled()
        backend.close()
    })
})
