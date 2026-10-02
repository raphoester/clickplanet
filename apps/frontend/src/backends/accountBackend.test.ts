import {afterEach, describe, expect, it, vi} from "vitest"
import {Code, ConnectError} from "@connectrpc/connect"
import {
    EmailRefusal,
    EmailRefusalReason,
    LinkRefusal,
    LinkRefusalReason,
    Provider as WireProvider,
    SignInIntent,
} from "../gen/grpc/auth/v1/auth_pb.ts"
import {AuthError} from "./account.ts"
import {ConnectAccountBackend} from "./accountBackend.ts"
import {newAuthServiceClient} from "./turnstileSession.ts"

const refusing = (code: Code) => vi.fn(async () => {
    throw new ConnectError("no", code)
})

function backendWith(methods: Record<string, unknown>, attest = vi.fn(async () => "turnstile-token")) {
    return new ConnectAccountBackend(methods as never, attest)
}

describe("ConnectAccountBackend", () => {
    it("reads the offered providers, and drops one this build has no name for", async () => {
        const backend = backendWith({
            getSignInOptions: vi.fn(async () => ({providers: [WireProvider.DISCORD, 7 as WireProvider, WireProvider.GOOGLE]})),
        })

        expect(await backend.signInOptions()).toEqual(["discord", "google"])
    })

    it("reads a server without the RPC as no provider", async () => {
        const backend = backendWith({getSignInOptions: refusing(Code.Unimplemented)})

        expect(await backend.signInOptions()).toEqual([])
    })

    it("reads a browser with no account as a guest", async () => {
        const backend = backendWith({getMe: refusing(Code.Unauthenticated)})

        expect(await backend.me()).toEqual({linked: []})
    })

    it("reads the linked providers", async () => {
        const backend = backendWith({getMe: vi.fn(async () => ({providers: [WireProvider.GOOGLE]}))})

        expect(await backend.me()).toEqual({linked: ["google"]})
    })

    it("asks for the provider on the wire and answers its URL", async () => {
        const startSignIn = vi.fn(async () => ({authorizationUrl: "https://discord.example/authorize"}))
        const backend = backendWith({startSignIn})

        expect(await backend.startSignIn("discord", "signIn")).toBe("https://discord.example/authorize")
        expect(startSignIn).toHaveBeenCalledWith({provider: WireProvider.DISCORD, intent: SignInIntent.SIGN_IN})
    })

    it("says on the wire when the trip is a link", async () => {
        const startSignIn = vi.fn(async () => ({authorizationUrl: "https://google.example/authorize"}))
        const backend = backendWith({startSignIn})

        await backend.startSignIn("google", "link")

        expect(startSignIn).toHaveBeenCalledWith({provider: WireProvider.GOOGLE, intent: SignInIntent.LINK})
    })

    it("reads a refused link from its detail", async () => {
        const cases: [LinkRefusalReason, string][] = [
            [LinkRefusalReason.IDENTITY_LINKED_ELSEWHERE, "linkedElsewhere"],
            [LinkRefusalReason.PROVIDER_ALREADY_LINKED, "alreadyLinked"],
        ]
        for (const [reason, failure] of cases) {
            const backend = backendWith({
                completeSignIn: vi.fn(async () => {
                    throw new ConnectError("no", Code.AlreadyExists, undefined, [new LinkRefusal({reason})])
                }),
            })

            await expect(backend.completeSignIn("code", "state")).rejects.toMatchObject({failure})
        }
    })

    it("maps each refusal to its failure", async () => {
        const cases: [Code, string][] = [
            [Code.Unimplemented, "off"],
            [Code.InvalidArgument, "notOffered"],
            [Code.ResourceExhausted, "tooManyTries"],
            [Code.FailedPrecondition, "startAgain"],
            [Code.PermissionDenied, "refused"],
            [Code.Unauthenticated, "notSignedIn"],
            [Code.Internal, "failed"],
        ]
        for (const [code, failure] of cases) {
            const backend = backendWith({completeSignIn: refusing(code)})

            const error = await backend.completeSignIn("code", "state").catch((e) => e)

            expect(error).toBeInstanceOf(AuthError)
            expect(error.failure).toBe(failure)
        }
    })

    it("reads an email identity as linked", async () => {
        const backend = backendWith({getMe: vi.fn(async () => ({providers: [WireProvider.EMAIL]}))})

        expect(await backend.me()).toEqual({linked: ["email"]})
    })

    it("asks for an email code with a fresh attestation each time", async () => {
        const startEmailSignIn = vi.fn(async () => ({}))
        const attest = vi.fn(async () => "turnstile-token")
        const backend = backendWith({startEmailSignIn}, attest)

        await backend.startEmailSignIn("player@example.com", "link")
        await backend.startEmailSignIn("player@example.com", "signIn")

        expect(attest).toHaveBeenCalledTimes(2)
        expect(startEmailSignIn).toHaveBeenCalledWith({
            email: "player@example.com", intent: SignInIntent.LINK, attestationToken: "turnstile-token",
        })
    })

    it("reads a refused address from its detail", async () => {
        const cases: [EmailRefusalReason, string][] = [
            [EmailRefusalReason.INVALID, "invalidEmail"],
            [EmailRefusalReason.DISPOSABLE, "disposableEmail"],
        ]
        for (const [reason, failure] of cases) {
            const backend = backendWith({
                startEmailSignIn: vi.fn(async () => {
                    throw new ConnectError("no", Code.InvalidArgument, undefined, [new EmailRefusal({reason})])
                }),
            })

            await expect(backend.startEmailSignIn("x", "signIn")).rejects.toMatchObject({failure})
        }
    })

    it("reads too many codes as such, and a widget that never answered as a failure", async () => {
        await expect(backendWith({startEmailSignIn: refusing(Code.ResourceExhausted)}).startEmailSignIn("x", "signIn"))
            .rejects.toMatchObject({failure: "tooManyCodes"})

        const startEmailSignIn = vi.fn(async () => ({}))
        const silent = backendWith({startEmailSignIn}, vi.fn(async () => {
            throw new Error("turnstile timed out")
        }))
        await expect(silent.startEmailSignIn("x", "signIn")).rejects.toMatchObject({failure: "failed"})
        expect(startEmailSignIn).not.toHaveBeenCalled()
    })

    it("reads a refused code by what the player can do about it", async () => {
        const cases: [Code, string][] = [
            [Code.InvalidArgument, "wrongCode"],
            [Code.FailedPrecondition, "newCode"],
            [Code.ResourceExhausted, "tooManyTries"],
            [Code.Unimplemented, "off"],
        ]
        for (const [code, failure] of cases) {
            const backend = backendWith({completeEmailSignIn: refusing(code)})

            await expect(backend.completeEmailSignIn("123456")).rejects.toMatchObject({failure})
        }
    })

    it("reads a refused email link from its detail", async () => {
        const backend = backendWith({
            completeEmailSignIn: vi.fn(async () => {
                throw new ConnectError("no", Code.AlreadyExists, undefined,
                    [new LinkRefusal({reason: LinkRefusalReason.IDENTITY_LINKED_ELSEWHERE})])
            }),
        })

        await expect(backend.completeEmailSignIn("123456")).rejects.toMatchObject({failure: "linkedElsewhere"})
    })

    it("sends a write once, even when the server cannot be reached", async () => {
        const completeSignIn = refusing(Code.Unavailable)
        const backend = backendWith({completeSignIn})

        await expect(backend.completeSignIn("code", "state")).rejects.toMatchObject({failure: "failed"})
        expect(completeSignIn).toHaveBeenCalledTimes(1)
    })
})

describe("the auth transport", () => {
    afterEach(() => {
        vi.unstubAllGlobals()
    })

    it("asks for the sign-in options with credentials", async () => {
        const fetch = vi.fn(async (): Promise<Response> => {
            throw new ConnectError("no", Code.Unimplemented)
        })
        vi.stubGlobal("fetch", fetch)

        await new ConnectAccountBackend(newAuthServiceClient({baseUrl: "https://api.test"}), async () => "").signInOptions()

        expect(String((fetch.mock.calls[0] as unknown[])[0])).toBe("https://api.test/auth.v1.AuthService/GetSignInOptions")
        expect(((fetch.mock.calls[0] as unknown[])[1] as RequestInit).credentials).toBe("include")
    })
})
