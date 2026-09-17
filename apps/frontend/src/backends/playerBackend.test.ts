import {afterEach, describe, expect, it, vi} from "vitest"
import {Code, ConnectError} from "@connectrpc/connect"
import {Profile as ProfilePb} from "../gen/grpc/player/v1/player_pb.ts"
import {isValidUsername, PlayerError} from "./player.ts"
import {ConnectPlayerBackend} from "./playerBackend.ts"
import {SESSION_HEADER, SessionProvider, SessionUnavailableError} from "./session.ts"

const refusing = (code: Code) => vi.fn(async () => {
    throw new ConnectError("no", code)
})

const answering = (name: string) => vi.fn(async () => ({profile: new ProfilePb({accountId: "account-1", name})}))

function sessionOf(...tokens: string[]): SessionProvider & {invalidate: ReturnType<typeof vi.fn>} {
    let minted = 0
    return {
        token: vi.fn(async () => tokens[Math.min(minted, tokens.length - 1)]),
        invalidate: vi.fn(() => {
            minted++
        }),
    }
}

function backendWith(methods: Record<string, unknown>, session: SessionProvider = sessionOf("token-1")) {
    return new ConnectPlayerBackend(methods as never, session)
}

const headersOf = (call: ReturnType<typeof vi.fn>, n = 0) =>
    (call.mock.calls[n] as unknown[])[1] as {headers: Headers}

afterEach(() => vi.restoreAllMocks())

describe("isValidUsername", () => {
    it("takes 3 to 20 letters, digits and underscores", () => {
        expect(isValidUsername("ana")).toBe(true)
        expect(isValidUsername("Ana_2024")).toBe(true)
        expect(isValidUsername("x".repeat(20))).toBe(true)
    })

    it("refuses a name too short or too long", () => {
        expect(isValidUsername("ab")).toBe(false)
        expect(isValidUsername("x".repeat(21))).toBe(false)
    })

    it("refuses anything but ASCII letters, digits and underscores", () => {
        expect(isValidUsername("ana bo")).toBe(false)
        expect(isValidUsername("anaïs")).toBe(false)
        expect(isValidUsername("ana-bo")).toBe(false)
    })

    // The chat puts that prefix before every guest's name.
    it("refuses a name that starts like a guest's, in any case", () => {
        expect(isValidUsername("guest_ana")).toBe(false)
        expect(isValidUsername("GUEST_ana")).toBe(false)
        expect(isValidUsername("guestana")).toBe(true)
    })
})

describe("ConnectPlayerBackend", () => {
    it("reads the profile with the click token", async () => {
        const getProfile = answering("ana")
        const backend = backendWith({getProfile})

        expect(await backend.profile()).toEqual({accountId: "account-1", name: "ana"})
        expect(headersOf(getProfile).headers.get(SESSION_HEADER)).toBe("token-1")
    })

    it("reads a profile with no name as an empty name", async () => {
        const backend = backendWith({getProfile: vi.fn(async () => ({}))})

        expect(await backend.profile()).toEqual({accountId: "", name: ""})
    })

    it("sends the name with the click token, and answers what the server stored", async () => {
        const setName = answering("Ana")
        const backend = backendWith({setName})

        expect(await backend.setName("Ana")).toEqual({accountId: "account-1", name: "Ana"})
        expect(setName).toHaveBeenCalledWith({name: "Ana"}, expect.anything())
        expect(headersOf(setName).headers.get(SESSION_HEADER)).toBe("token-1")
    })

    // The token may name the account the browser was on before a sign-in.
    it("sends a refused call once more with a fresh session", async () => {
        const session = sessionOf("stale", "fresh")
        const setName = vi.fn()
            .mockRejectedValueOnce(new ConnectError("no", Code.Unauthenticated))
            .mockResolvedValue({profile: new ProfilePb({name: "ana"})})
        const backend = backendWith({setName}, session)

        expect(await backend.setName("ana")).toMatchObject({name: "ana"})
        expect(session.invalidate).toHaveBeenCalledTimes(1)
        expect(headersOf(setName, 0).headers.get(SESSION_HEADER)).toBe("stale")
        expect(headersOf(setName, 1).headers.get(SESSION_HEADER)).toBe("fresh")
    })

    it("reports a second refusal as not signed in, and does not loop", async () => {
        const getProfile = refusing(Code.Unauthenticated)
        const backend = backendWith({getProfile})

        await expect(backend.profile()).rejects.toMatchObject({failure: "notSignedIn"})
        expect(getProfile).toHaveBeenCalledTimes(2)
    })

    it("maps each refusal to its failure", async () => {
        const cases: [Code, string][] = [
            [Code.InvalidArgument, "invalid"],
            [Code.AlreadyExists, "taken"],
            [Code.PermissionDenied, "guest"],
            [Code.Internal, "failed"],
        ]
        for (const [code, failure] of cases) {
            const backend = backendWith({setName: refusing(code)})

            const error = await backend.setName("ana").catch((e) => e)

            expect(error).toBeInstanceOf(PlayerError)
            expect(error.failure).toBe(failure)
        }
    })

    it("reports a session that could not be minted as failed", async () => {
        const session: SessionProvider = {
            token: vi.fn(async () => {
                throw new SessionUnavailableError()
            }),
            invalidate: vi.fn(),
        }
        const setName = answering("ana")
        const backend = backendWith({setName}, session)

        await expect(backend.setName("ana")).rejects.toMatchObject({failure: "failed"})
        expect(setName).not.toHaveBeenCalled()
    })

    it("retries the read while the server cannot be reached", async () => {
        vi.spyOn(console, "error").mockImplementation(() => {})
        const getProfile = vi.fn()
            .mockRejectedValueOnce(new ConnectError("down", Code.Unavailable))
            .mockResolvedValue({profile: new ProfilePb({name: "ana"})})
        const backend = backendWith({getProfile})

        expect(await backend.profile()).toMatchObject({name: "ana"})
        expect(getProfile).toHaveBeenCalledTimes(2)
    })

    it("sends a write once, even when the server cannot be reached", async () => {
        const setName = refusing(Code.Unavailable)
        const backend = backendWith({setName})

        await expect(backend.setName("ana")).rejects.toMatchObject({failure: "failed"})
        expect(setName).toHaveBeenCalledTimes(1)
    })
})
