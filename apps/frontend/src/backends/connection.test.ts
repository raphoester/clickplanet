import {describe, expect, it} from "vitest"
import {Code, ConnectError, StreamResponse, UnaryRequest, UnaryResponse} from "@connectrpc/connect"
import {Connection, ConnectionHealth, connectionInterceptor, DOWN_AFTER_MISSES, followBrowser} from "./connection.ts"

function watched(health: ConnectionHealth): Connection[] {
    const seen: Connection[] = []
    health.watchConnection((connection) => seen.push(connection))
    return seen
}

function request(signal: AbortSignal = new AbortController().signal): UnaryRequest {
    return {stream: false, signal} as unknown as UnaryRequest
}

function answering(): (req: unknown) => Promise<UnaryResponse> {
    return async () => ({stream: false}) as unknown as UnaryResponse
}

function failing(failure: unknown): (req: unknown) => Promise<UnaryResponse> {
    return async () => {
        throw failure
    }
}

async function call(health: ConnectionHealth, next: (req: unknown) => Promise<UnaryResponse | StreamResponse>, req = request()) {
    return connectionInterceptor(health)(next)(req)
}

async function miss(health: ConnectionHealth) {
    await call(health, failing(new TypeError("Failed to fetch"))).catch(() => {})
}

describe("ConnectionHealth", () => {
    it("tells a new watcher where it stands", () => {
        expect(watched(new ConnectionHealth())).toEqual(["up"])
    })

    it("goes down on misses in a row, and one alone is a blip", () => {
        const health = new ConnectionHealth()
        const seen = watched(health)

        health.missed()
        expect(seen).toEqual(["up"])

        for (let i = 1; i < DOWN_AFTER_MISSES; i++) health.missed()
        expect(seen).toEqual(["up", "down"])
    })

    it("comes back on the first answer, and says each change once", () => {
        const health = new ConnectionHealth()
        const seen = watched(health)

        for (let i = 0; i < DOWN_AFTER_MISSES + 3; i++) health.missed()
        health.reached()
        health.reached()

        expect(seen).toEqual(["up", "down", "up"])
    })

    it("counts the misses afresh after an answer", () => {
        const health = new ConnectionHealth()
        const seen = watched(health)

        health.missed()
        health.reached()
        health.missed()

        expect(seen).toEqual(["up"])
    })

    it("is down at once when the browser says it is offline", () => {
        const health = new ConnectionHealth()
        const seen = watched(health)
        const browser = new EventTarget()
        const stop = followBrowser(health, browser)

        browser.dispatchEvent(new Event("offline"))
        expect(seen).toEqual(["up", "down"])

        stop()
        health.reached()
        browser.dispatchEvent(new Event("offline"))
        expect(seen).toEqual(["up", "down", "up"])
    })
})

describe("connectionInterceptor", () => {
    it("reads an answer as the server reached", async () => {
        const health = new ConnectionHealth()
        const seen = watched(health)
        health.missed()
        health.missed()

        await call(health, answering())

        expect(seen).toEqual(["up", "down", "up"])
    })

    it("reads a refusal the server sent as reached", async () => {
        const health = new ConnectionHealth()
        const seen = watched(health)
        health.missed()
        health.missed()

        await expect(call(health, failing(new ConnectError("too many clicks", Code.ResourceExhausted)))).rejects.toThrow()

        expect(seen).toEqual(["up", "down", "up"])
    })

    it("reads a failed fetch as a miss", async () => {
        const health = new ConnectionHealth()
        const seen = watched(health)

        for (let i = 0; i < DOWN_AFTER_MISSES; i++) {
            await expect(call(health, failing(new TypeError("Failed to fetch")))).rejects.toThrow()
        }

        expect(seen).toEqual(["up", "down"])
    })

    it("reads a timeout as a miss", async () => {
        const health = new ConnectionHealth()
        const seen = watched(health)

        for (let i = 0; i < DOWN_AFTER_MISSES; i++) {
            const timer = new AbortController()
            timer.abort(new ConnectError("the operation timed out", Code.DeadlineExceeded))
            await expect(call(health, failing(new DOMException("aborted", "AbortError")), request(timer.signal))).rejects.toThrow()
        }

        expect(seen).toEqual(["up", "down"])
    })

    it("reads a proxy with no server behind it as a miss", async () => {
        const health = new ConnectionHealth()
        const seen = watched(health)

        for (let i = 0; i < DOWN_AFTER_MISSES; i++) {
            await expect(call(health, failing(new ConnectError("bad gateway", Code.Unavailable)))).rejects.toThrow()
        }

        expect(seen).toEqual(["up", "down"])
    })

    it("says nothing about a call the page cancelled", async () => {
        const health = new ConnectionHealth()
        const seen = watched(health)
        health.missed()

        const page = new AbortController()
        page.abort()
        await expect(call(health, failing(new DOMException("aborted", "AbortError")), request(page.signal))).rejects.toThrow()

        health.missed()
        expect(seen).toEqual(["up", "down"])
    })

    it("follows a stream past its opening, and reads its break as a miss", async () => {
        const health = new ConnectionHealth()
        const seen = watched(health)
        await miss(health)

        async function* messages() {
            yield "heartbeat"
            throw new TypeError("network error")
        }
        const res = await call(health, async () => ({stream: true, message: messages()}) as unknown as StreamResponse)
        expect(seen).toEqual(["up"])

        const received: unknown[] = []
        await expect((async () => {
            for await (const message of (res as StreamResponse).message) received.push(message)
        })()).rejects.toThrow()
        await miss(health)

        expect(received).toEqual(["heartbeat"])
        expect(seen).toEqual(["up", "down"])
    })
})
