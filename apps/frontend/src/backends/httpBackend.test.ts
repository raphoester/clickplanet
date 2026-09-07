import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {ClickServiceClient, decodeTileUpdate, openUpdatesSocket, websocketUrl} from "./httpBackend.ts"
import {TileUpdate} from "../gen/grpc/clicks/v1/clicks_pb.ts"
import type {Update} from "./backend.ts"

function frame(update: Partial<{tileId: number, countryId: string, previousCountryId: string}>): ArrayBuffer {
    const bytes = new TileUpdate(update).toBinary()
    return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength) as ArrayBuffer
}

describe("websocketUrl", () => {
    it("swaps the scheme and keeps the host", () => {
        expect(websocketUrl("https://api.clickplanet.lol")).toBe("wss://api.clickplanet.lol/v2/ws/listen")
        expect(websocketUrl("http://localhost:8080")).toBe("ws://localhost:8080/v2/ws/listen")
    })

    /** The old version string-replaced "https://" away, so a host containing it broke. */
    it("only rewrites the leading scheme", () => {
        expect(websocketUrl("https://api.http://x.dev")).toBe("wss://api.http://x.dev/v2/ws/listen")
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

describe("ClickServiceClient.fetch", () => {
    afterEach(() => vi.unstubAllGlobals())

    const client = () => new ClickServiceClient({baseUrl: "https://api.test", timeoutMs: 50})

    it("returns undefined when the response carries no payload", async () => {
        vi.stubGlobal("fetch", vi.fn(async () => new Response(JSON.stringify({}))))
        await expect(client().fetch("POST", "/x")).resolves.toBeUndefined()
    })

    it("decodes the base64 payload", async () => {
        vi.stubGlobal("fetch", vi.fn(async () => new Response(JSON.stringify({data: btoa("hi")}))))
        await expect(client().fetch("POST", "/x")).resolves.toEqual(new Uint8Array([104, 105]))
    })

    it("retries a network failure and succeeds", async () => {
        vi.spyOn(console, "error").mockImplementation(() => {})
        const stub = vi.fn()
            .mockRejectedValueOnce(new Error("offline"))
            .mockResolvedValueOnce(new Response(JSON.stringify({data: btoa("ok")})))
        vi.stubGlobal("fetch", stub)

        await expect(client().fetch("POST", "/x")).resolves.toEqual(new Uint8Array([111, 107]))
        expect(stub).toHaveBeenCalledTimes(2)
    })

    it("gives up after five attempts", async () => {
        vi.spyOn(console, "error").mockImplementation(() => {})
        const stub = vi.fn().mockRejectedValue(new Error("offline"))
        vi.stubGlobal("fetch", stub)

        await expect(client().fetch("POST", "/x")).rejects.toThrow(/after 5 attempts/)
        expect(stub).toHaveBeenCalledTimes(5)
    })

    /** A 4xx/5xx is the server answering, so retrying it just multiplies the load. */
    it("does not retry a rejected request", async () => {
        const stub = vi.fn(async () => new Response("nope", {status: 500, statusText: "Server Error"}))
        vi.stubGlobal("fetch", stub)

        await expect(client().fetch("POST", "/x")).rejects.toThrow(/500/)
        expect(stub).toHaveBeenCalledTimes(1)
    })

    it("stops retrying once the caller aborts", async () => {
        vi.spyOn(console, "error").mockImplementation(() => {})
        const controller = new AbortController()
        const stub = vi.fn(async () => {
            controller.abort()
            throw new Error("offline")
        })
        vi.stubGlobal("fetch", stub)

        await expect(client().fetch("POST", "/x", undefined, controller.signal)).rejects.toThrow()
        expect(stub).toHaveBeenCalledTimes(1)
    })
})
