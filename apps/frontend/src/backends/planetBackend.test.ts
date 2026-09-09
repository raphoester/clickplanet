import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {bindingsOf, decodeTileUpdate, openUpdatesSocket, PlanetBackend, websocketUrl} from "./planetBackend.ts"
import {Code, ConnectError} from "@connectrpc/connect"
import {GetMapResponse, TileUpdate} from "../gen/grpc/planet/v1/planet_pb.ts"
import {RateLimitedError, type Update, VPNBlockedError} from "./backend.ts"
import {SESSION_HEADER, type SessionProvider, SessionUnavailableError} from "./session.ts"

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

function frame(update: Partial<{tileId: number, countryId: string, previousCountryId: string}>): ArrayBuffer {
    const bytes = new TileUpdate(update).toBinary()
    return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength) as ArrayBuffer
}

describe("websocketUrl", () => {
    it("swaps the scheme and keeps the host", () => {
        expect(websocketUrl("https://api.clickplanet.lol")).toBe("wss://api.clickplanet.lol/ws/listen")
        expect(websocketUrl("http://localhost:8080")).toBe("ws://localhost:8080/ws/listen")
    })

    it("only rewrites the leading scheme", () => {
        expect(websocketUrl("https://api.http://x.dev")).toBe("wss://api.http://x.dev/ws/listen")
    })
})

describe("decodeTileUpdate", () => {
    it("decodes a tile update frame", () => {
        expect(decodeTileUpdate(frame({tileId: 7, countryId: "jp", previousCountryId: "fr"})))
            .toEqual({tile: 7, previousCountry: "fr", newCountry: "jp"})
    })

    it("reports an unowned previous tile as undefined rather than an empty code", () => {
        expect(decodeTileUpdate(frame({tileId: 1, countryId: "fr"})))
            .toEqual({tile: 1, previousCountry: undefined, newCountry: "fr"})
    })

    it("drops a frame it cannot parse", () => {
        vi.spyOn(console, "error").mockImplementation(() => {})
        expect(decodeTileUpdate(new Uint8Array([0xff, 0xff, 0xff, 0xff]).buffer)).toBeUndefined()
        expect(decodeTileUpdate("not binary")).toBeUndefined()
    })
})

class FakeWebSocket {
    static instances: FakeWebSocket[] = []

    binaryType = ""
    onopen: (() => void) | null = null
    onmessage: ((event: {data: unknown}) => void) | null = null
    onclose: (() => void) | null = null
    closed = false

    constructor(public url: string) {
        FakeWebSocket.instances.push(this)
    }

    close() {
        this.closed = true
        this.onclose?.()
    }

    drop() {
        this.onclose?.()
    }
}

describe("openUpdatesSocket", () => {
    beforeEach(() => {
        vi.useFakeTimers()
        FakeWebSocket.instances = []
        vi.stubGlobal("WebSocket", FakeWebSocket)
    })

    afterEach(() => {
        vi.useRealTimers()
        vi.unstubAllGlobals()
    })

    const latest = () => FakeWebSocket.instances[FakeWebSocket.instances.length - 1]

    it("connects and forwards decoded updates", () => {
        const received: Update[] = []
        openUpdatesSocket("wss://example.test/ws", u => received.push(u))

        expect(FakeWebSocket.instances).toHaveLength(1)
        expect(latest().url).toBe("wss://example.test/ws")
        expect(latest().binaryType).toBe("arraybuffer")

        latest().onmessage!({data: frame({tileId: 3, countryId: "de"})})
        expect(received).toEqual([{tile: 3, previousCountry: undefined, newCountry: "de"}])
    })

    it("does not forward a frame it could not decode", () => {
        vi.spyOn(console, "error").mockImplementation(() => {})
        const received: Update[] = []
        openUpdatesSocket("wss://example.test/ws", u => received.push(u))

        latest().onmessage!({data: "junk"})
        expect(received).toEqual([])
    })

    it("reconnects after the connection drops", () => {
        openUpdatesSocket("wss://example.test/ws", () => {})
        expect(FakeWebSocket.instances).toHaveLength(1)

        latest().drop()
        vi.advanceTimersByTime(500)
        expect(FakeWebSocket.instances).toHaveLength(2)
    })

    it("backs off exponentially while the backend stays down", () => {
        openUpdatesSocket("wss://example.test/ws", () => {})

        const delays = [500, 1000, 2000, 4000]
        for (const [i, delay] of delays.entries()) {
            latest().drop()
            vi.advanceTimersByTime(delay - 1)
            expect(FakeWebSocket.instances, `retry ${i} fired early`).toHaveLength(i + 1)
            vi.advanceTimersByTime(1)
            expect(FakeWebSocket.instances, `retry ${i} did not fire`).toHaveLength(i + 2)
        }
    })

    it("caps the backoff", () => {
        openUpdatesSocket("wss://example.test/ws", () => {})
        for (let i = 0; i < 20; i++) {
            latest().drop()
            vi.advanceTimersByTime(30_000)
        }
        const before = FakeWebSocket.instances.length
        latest().drop()
        vi.advanceTimersByTime(30_000)
        expect(FakeWebSocket.instances).toHaveLength(before + 1)
    })

    it("resets the backoff once a connection succeeds", () => {
        openUpdatesSocket("wss://example.test/ws", () => {})

        latest().drop()
        vi.advanceTimersByTime(500)
        latest().drop()
        vi.advanceTimersByTime(1000)

        latest().onopen!()

        const before = FakeWebSocket.instances.length
        latest().drop()
        vi.advanceTimersByTime(500)
        expect(FakeWebSocket.instances).toHaveLength(before + 1)
    })

    it("closes the socket when the caller stops listening", () => {
        const close = openUpdatesSocket("wss://example.test/ws", () => {})
        const socket = latest()

        close()
        expect(socket.closed).toBe(true)
    })

    it("does not reconnect after the caller stops listening", () => {
        const close = openUpdatesSocket("wss://example.test/ws", () => {})
        close()

        vi.advanceTimersByTime(60_000)
        expect(FakeWebSocket.instances).toHaveLength(1)
    })

    it("cancels a retry that was already scheduled", () => {
        const close = openUpdatesSocket("wss://example.test/ws", () => {})
        latest().drop()
        close()

        vi.advanceTimersByTime(60_000)
        expect(FakeWebSocket.instances).toHaveLength(1)
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

type GetMapRequestFields = {startTileId: number, endTileId: number}
type GetMapMock = ReturnType<typeof getMapMock>

function getMapMock(impl?: (req: GetMapRequestFields) => Promise<GetMapResponse>) {
    return vi.fn<(req: GetMapRequestFields) => Promise<GetMapResponse>>(impl)
}

describe("PlanetBackend.getCurrentOwnershipsByBatch", () => {
    const backendWith = (getMap: GetMapMock) => {
        const client = {click: vi.fn(), getMap, mapDensity: vi.fn()} as never
        return new PlanetBackend({baseUrl: "https://api.test"}, client, 1_000)
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
        const clientStub = {click, getMap: vi.fn(), mapDensity: vi.fn()} as never
        return new PlanetBackend({baseUrl: "https://api.test"}, clientStub, 1_000, session)
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
