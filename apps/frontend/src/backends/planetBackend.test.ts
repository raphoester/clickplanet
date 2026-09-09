import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {bindingsOf, decodeTileUpdate, openUpdatesSocket, PlanetBackend, websocketUrl} from "./planetBackend.ts"
import {Code, ConnectError} from "@connectrpc/connect"
import {GetMapResponse, TileUpdate} from "../gen/grpc/planet/v1/planet_pb.ts"
import type {Update} from "./backend.ts"

function frame(update: Partial<{tileId: number, countryId: string, previousCountryId: string}>): ArrayBuffer {
    const bytes = new TileUpdate(update).toBinary()
    return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength) as ArrayBuffer
}

describe("websocketUrl", () => {
    it("swaps the scheme and keeps the host", () => {
        expect(websocketUrl("https://api.clickplanet.lol")).toBe("wss://api.clickplanet.lol/ws/listen")
        expect(websocketUrl("http://localhost:8080")).toBe("ws://localhost:8080/ws/listen")
    })

    /** The old version string-replaced "https://" away, so a host containing it broke. */
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

    /** A bad frame must be dropped: it used to throw out of the socket's onmessage. */
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

    /** Simulates the server or network dropping the connection. */
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

    /**
     * The socket is the only source of live tile changes. Before this, a drop
     * was never retried and the globe silently froze until a reload.
     */
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

        latest().onopen!() // reconnected

        const before = FakeWebSocket.instances.length
        latest().drop()
        vi.advanceTimersByTime(500)
        expect(FakeWebSocket.instances).toHaveLength(before + 1)
    })

    /** The returned closer used to reference `websocket.close` without calling it. */
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

    /** Unowned tiles are left out, which is the shape the globe already applies. */
    it("skips unowned tiles rather than binding them to an empty code", () => {
        expect(bindingsOf(mapResponse(0, ["", "fr"], [0, 1, 0]))).toEqual(new Map([[1, "fr"]]))
    })

    it("reads codes longer than two characters", () => {
        expect(bindingsOf(mapResponse(0, ["", "gb-eng"], [1]))).toEqual(new Map([[0, "gb-eng"]]))
    })

    it("handles an empty range", () => {
        expect(bindingsOf(mapResponse(0, [""], []))).toEqual(new Map())
    })

    /** A view onto a larger buffer must not read its neighbours. */
    it("survives tiles that do not start at offset zero", () => {
        const padded = new Uint8Array([0xff, 0xff, ...tileBytes([1])])
        const res = new GetMapResponse({startTileId: 4, codes: ["", "jp"], tiles: padded.subarray(2)})
        expect(bindingsOf(res)).toEqual(new Map([[4, "jp"]]))
    })
})

type GetMapRequestFields = {startTileId: number, endTileId: number}
type GetMapMock = ReturnType<typeof getMapMock>

/** Typed so `mock.calls` keeps its argument, which `vi.fn()` alone loses. */
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

    /** An answer the server chose to send is never retried: that only adds load. */
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
