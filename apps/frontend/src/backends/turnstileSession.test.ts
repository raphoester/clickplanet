import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {Code, ConnectError} from "@connectrpc/connect"
import {newAuthServiceClient, SessionClient} from "./turnstileSession.ts"
import {newClickServiceClient} from "./planetBackend.ts"
import {SessionUnavailableError} from "./session.ts"

const HOUR_MS = 60 * 60 * 1000

let now = 1_000_000

function clock() {
    return now
}

type CreateSession = (req: {attestationToken: string}) => Promise<{token: string, expiresAtUnixMs: bigint}>

function fakeClient(createSession: CreateSession) {
    return {createSession: vi.fn(createSession)} as never
}

function minting(token: string, ttlMs = HOUR_MS) {
    return async () => ({token, expiresAtUnixMs: BigInt(now + ttlMs)})
}

beforeEach(() => {
    now = 1_000_000
})

describe("SessionClient", () => {
    it("mints once and reuses the token until it is close to lapsing", async () => {
        const createSession = vi.fn(minting("session-1"))
        const attest = vi.fn(async () => "widget-token")

        const client = new SessionClient(fakeClient(createSession), attest, {now: clock})

        expect(await client.token()).toBe("session-1")
        expect(await client.token()).toBe("session-1")

        now += 30 * 60 * 1000
        expect(await client.token()).toBe("session-1")

        expect(createSession).toHaveBeenCalledTimes(1)
        expect(attest).toHaveBeenCalledTimes(1)
    })

    it("passes the attestation token to the server", async () => {
        const createSession = vi.fn(minting("session-1"))

        const client = new SessionClient(fakeClient(createSession), async () => "widget-token", {now: clock})
        await client.token()

        expect(createSession).toHaveBeenCalledWith({attestationToken: "widget-token"})
    })

    // Minting inside the margin rather than after the expiry: a token that
    // lapses between the check and the server reading it costs a round trip and
    // a retry that the player can feel.
    it("mints again once the token is inside the refresh margin", async () => {
        const createSession = vi.fn()
            .mockImplementationOnce(minting("session-1"))
            .mockImplementationOnce(minting("session-2"))

        const client = new SessionClient(fakeClient(createSession), async () => "widget-token", {
            now: clock,
            refreshMarginMs: 60_000,
        })

        expect(await client.token()).toBe("session-1")

        now += HOUR_MS - 61_000
        expect(await client.token()).toBe("session-1")

        now += 2_000
        expect(await client.token()).toBe("session-2")
        expect(createSession).toHaveBeenCalledTimes(2)
    })

    it("mints again after the server refused the token it gave us", async () => {
        const createSession = vi.fn()
            .mockImplementationOnce(minting("session-1"))
            .mockImplementationOnce(minting("session-2"))

        const client = new SessionClient(fakeClient(createSession), async () => "widget-token", {now: clock})

        expect(await client.token()).toBe("session-1")
        client.invalidate()
        expect(await client.token()).toBe("session-2")
    })

    // A sign-in or a sign-out changes the account the cookie names while a
    // mint may be in flight. That mint names the old account, so it must not
    // be kept for the clicks after it.
    it("does not keep a mint that started before an invalidation", async () => {
        let release: (value: {token: string, expiresAtUnixMs: bigint}) => void = () => {}
        const createSession = vi.fn()
            .mockImplementationOnce(() => new Promise((resolve) => {
                release = resolve
            }))
            .mockImplementationOnce(minting("session-2"))

        const client = new SessionClient(fakeClient(createSession), async () => "widget-token", {now: clock})

        const old = client.token()
        await vi.waitFor(() => expect(createSession).toHaveBeenCalledTimes(1))
        client.invalidate()
        release({token: "session-1", expiresAtUnixMs: BigInt(now + HOUR_MS)})
        await old

        expect(await client.token()).toBe("session-2")
        expect(await client.token()).toBe("session-2")
        expect(createSession).toHaveBeenCalledTimes(2)
    })

    // A page load fires a flurry of clicks. One mint has to serve all of them,
    // or the first second of play spends the whole per-IP mint budget.
    it("serves concurrent callers from a single mint", async () => {
        let release: (value: {token: string, expiresAtUnixMs: bigint}) => void = () => {}
        const inFlight = new Promise<{token: string, expiresAtUnixMs: bigint}>((resolve) => {
            release = resolve
        })

        const createSession = vi.fn(() => inFlight)
        const attest = vi.fn(async () => "widget-token")

        const client = new SessionClient(fakeClient(createSession), attest, {now: clock})

        const tokens = Promise.all([client.token(), client.token(), client.token()])
        release({token: "session-1", expiresAtUnixMs: BigInt(now + HOUR_MS)})

        expect(await tokens).toEqual(["session-1", "session-1", "session-1"])
        expect(createSession).toHaveBeenCalledTimes(1)
        expect(attest).toHaveBeenCalledTimes(1)
    })

    it("can mint again after a failure rather than being stuck on it", async () => {
        const createSession = vi.fn()
            .mockImplementationOnce(async () => {
                throw new ConnectError("no", Code.PermissionDenied)
            })
            .mockImplementationOnce(minting("session-1"))

        const client = new SessionClient(fakeClient(createSession), async () => "widget-token", {now: clock})

        await expect(client.token()).rejects.toBeInstanceOf(SessionUnavailableError)
        expect(await client.token()).toBe("session-1")
    })

    // A refused mint answers permission_denied, exactly as a VPN-blocked click
    // does. Left bare it would reach the dialog telling the player to turn off a
    // VPN they may not be using.
    it("reports a refused mint as a session failure, not as the code it answered", async () => {
        const createSession = vi.fn(async () => {
            throw new ConnectError("refused", Code.PermissionDenied)
        })

        const client = new SessionClient(fakeClient(createSession), async () => "widget-token", {now: clock})

        await expect(client.token()).rejects.toBeInstanceOf(SessionUnavailableError)
    })

    it("reports a failed attestation the same way, without calling the server", async () => {
        const createSession = vi.fn(minting("session-1"))
        const attest = vi.fn(async () => {
            throw new Error("Turnstile did not answer in time")
        })

        const client = new SessionClient(fakeClient(createSession), attest, {now: clock})

        await expect(client.token()).rejects.toBeInstanceOf(SessionUnavailableError)
        expect(createSession).not.toHaveBeenCalled()
    })

    describe("held", () => {
        // Presence asks this on a timer: a mint here would put a Turnstile
        // check behind every visitor who never clicked.
        it("holds nothing before the first mint, and does not start one", () => {
            const createSession = vi.fn(minting("session-1"))
            const attest = vi.fn(async () => "widget-token")

            const client = new SessionClient(fakeClient(createSession), attest, {now: clock})

            expect(client.held()).toBeUndefined()
            expect(attest).not.toHaveBeenCalled()
            expect(createSession).not.toHaveBeenCalled()
        })

        it("holds the token a click minted", async () => {
            const client = new SessionClient(fakeClient(minting("session-1")), async () => "widget-token", {now: clock})

            await client.token()

            expect(client.held()).toBe("session-1")
        })

        it("lets go of it inside the refresh margin, as token does", async () => {
            const client = new SessionClient(fakeClient(minting("session-1")), async () => "widget-token", {
                now: clock,
                refreshMarginMs: 60_000,
            })
            await client.token()

            now += HOUR_MS - 59_000

            expect(client.held()).toBeUndefined()
        })

        it("lets go of it on an invalidation", async () => {
            const client = new SessionClient(fakeClient(minting("session-1")), async () => "widget-token", {now: clock})
            await client.token()

            client.invalidate()

            expect(client.held()).toBeUndefined()
        })
    })
})

/** A fetch that records what it was asked and answers nothing a client can read. */
function recordingFetch() {
    const fetch = vi.fn(async (...args: Parameters<typeof globalThis.fetch>): Promise<Response> => {
        void args
        throw new TypeError("offline")
    })
    vi.stubGlobal("fetch", fetch)
    return fetch
}

describe("the API transports", () => {
    afterEach(() => {
        vi.unstubAllGlobals()
    })

    // The account lives in an HttpOnly cookie on the API's host. A cross-origin
    // mint neither sends it nor keeps the one the answer sets unless it asks for
    // credentials, and every mint would then start a new guest.
    it("mints with credentials, so the account cookie travels", async () => {
        const fetch = recordingFetch()

        await expect(newAuthServiceClient({baseUrl: "https://api.test"}).createSession({attestationToken: "t"}))
            .rejects.toThrow()

        expect(fetch).toHaveBeenCalledTimes(1)
        expect(String(fetch.mock.calls[0][0])).toBe("https://api.test/auth.v1.AuthService/CreateSession")
        expect(fetch.mock.calls[0][1]?.credentials).toBe("include")
    })

    // Only the mint needs to know who is asking. A click or a map read that
    // carries a cookie is one no shared cache serves.
    it("clicks without credentials", async () => {
        const fetch = recordingFetch()

        await expect(newClickServiceClient({baseUrl: "https://api.test"}).click({tileId: 1, countryId: "fr"}))
            .rejects.toThrow()

        expect(fetch).toHaveBeenCalledTimes(1)
        expect(fetch.mock.calls[0][1]?.credentials).toBe("same-origin")
    })
})
