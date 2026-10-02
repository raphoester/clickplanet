import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {Code, ConnectError} from "@connectrpc/connect"
import {
    HeldSession,
    localTokenStore,
    newAuthServiceClient,
    SESSION_STORAGE_KEY,
    SessionClient,
} from "./turnstileSession.ts"
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

function fakeStore(kept?: HeldSession) {
    let held = kept

    return {
        read: vi.fn(() => held),
        write: vi.fn((session: HeldSession) => {
            held = session
        }),
        clear: vi.fn(() => {
            held = undefined
        }),
    }
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

    describe("a token kept from the last page load", () => {
        it("holds it at once, with no mint and no attestation", () => {
            const createSession = vi.fn(minting("session-2"))
            const attest = vi.fn(async () => "widget-token")
            const store = fakeStore({value: "session-1", expiresAt: now + HOUR_MS})

            const client = new SessionClient(fakeClient(createSession), attest, {now: clock, store})

            expect(client.held()).toBe("session-1")
            expect(createSession).not.toHaveBeenCalled()
            expect(attest).not.toHaveBeenCalled()
        })

        it("leaves one inside the refresh margin, by the rule held applies", async () => {
            const store = fakeStore({value: "session-1", expiresAt: now + 59_000})

            const client = new SessionClient(fakeClient(minting("session-2")), async () => "widget-token", {
                now: clock,
                refreshMarginMs: 60_000,
                store,
            })

            expect(client.held()).toBeUndefined()
            expect(await client.token()).toBe("session-2")
        })

        it("keeps what it mints, for the next page load", async () => {
            const store = fakeStore()

            const client = new SessionClient(fakeClient(minting("session-1")), async () => "widget-token", {
                now: clock,
                store,
            })
            await client.token()

            expect(store.write).toHaveBeenCalledWith({value: "session-1", expiresAt: now + HOUR_MS})
        })

        it("drops it on an invalidation", async () => {
            const store = fakeStore()
            const client = new SessionClient(fakeClient(minting("session-1")), async () => "widget-token", {
                now: clock,
                store,
            })
            await client.token()

            client.invalidate()

            expect(store.read()).toBeUndefined()
        })

        it("drops it when a mint fails, so nothing is kept that is not held", async () => {
            const store = fakeStore({value: "session-1", expiresAt: now - 1})
            const createSession = vi.fn(async () => {
                throw new ConnectError("no", Code.PermissionDenied)
            })

            const client = new SessionClient(fakeClient(createSession), async () => "widget-token", {now: clock, store})

            await expect(client.token()).rejects.toBeInstanceOf(SessionUnavailableError)
            expect(store.read()).toBeUndefined()
        })
    })
})

function stubStorage(kept: Record<string, string> = {}) {
    vi.stubGlobal("window", {
        localStorage: {
            getItem: (key: string) => kept[key] ?? null,
            setItem: (key: string, value: string) => {
                kept[key] = value
            },
            removeItem: (key: string) => {
                delete kept[key]
            },
        },
    })

    return kept
}

describe("localTokenStore", () => {
    afterEach(() => {
        vi.unstubAllGlobals()
    })

    it("reads back what it wrote, and lets go of it", () => {
        stubStorage()
        const store = localTokenStore()

        store.write({value: "session-1", expiresAt: 42})
        expect(store.read()).toEqual({value: "session-1", expiresAt: 42})

        store.clear()
        expect(store.read()).toBeUndefined()
    })

    it("reads nothing when nothing was kept", () => {
        stubStorage()

        expect(localTokenStore().read()).toBeUndefined()
    })

    it("reads nothing from something that is not a kept token", () => {
        stubStorage({[SESSION_STORAGE_KEY]: '{"value": 5}'})

        expect(localTokenStore().read()).toBeUndefined()
    })

    it("reads nothing, and neither writing nor clearing throws, when storage refuses", () => {
        const refuse = () => {
            throw new Error("the storage is not available")
        }
        vi.stubGlobal("window", {localStorage: {getItem: refuse, setItem: refuse, removeItem: refuse}})

        const store = localTokenStore()

        expect(store.read()).toBeUndefined()
        expect(() => store.write({value: "session-1", expiresAt: 42})).not.toThrow()
        expect(() => store.clear()).not.toThrow()
    })
})

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

    it("mints with credentials, so the account cookie travels", async () => {
        const fetch = recordingFetch()

        await expect(newAuthServiceClient({baseUrl: "https://api.test"}).createSession({attestationToken: "t"}))
            .rejects.toThrow()

        expect(fetch).toHaveBeenCalledTimes(1)
        expect(String(fetch.mock.calls[0][0])).toBe("https://api.test/auth.v1.AuthService/CreateSession")
        expect(fetch.mock.calls[0][1]?.credentials).toBe("include")
    })

    it("clicks without credentials", async () => {
        const fetch = recordingFetch()

        await expect(newClickServiceClient({baseUrl: "https://api.test"}).click({tileId: 1, countryId: "fr"}))
            .rejects.toThrow()

        expect(fetch).toHaveBeenCalledTimes(1)
        expect(fetch.mock.calls[0][1]?.credentials).toBe("same-origin")
    })
})
