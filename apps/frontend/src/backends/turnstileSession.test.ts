import {beforeEach, describe, expect, it, vi} from "vitest"
import {Code, ConnectError} from "@connectrpc/connect"
import {SessionClient} from "./turnstileSession.ts"
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
})
