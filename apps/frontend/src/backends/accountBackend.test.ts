import {afterEach, describe, expect, it, vi} from "vitest"
import {Code, ConnectError} from "@connectrpc/connect"
import {Provider as WireProvider} from "../gen/grpc/auth/v1/auth_pb.ts"
import {AuthError} from "./account.ts"
import {ConnectAccountBackend} from "./accountBackend.ts"
import {newAuthServiceClient} from "./turnstileSession.ts"

const refusing = (code: Code) => vi.fn(async () => {
    throw new ConnectError("no", code)
})

function backendWith(methods: Record<string, unknown>) {
    return new ConnectAccountBackend(methods as never)
}

describe("ConnectAccountBackend", () => {
    it("reads the offered providers, and drops one this build has no name for", async () => {
        const backend = backendWith({
            getSignInOptions: vi.fn(async () => ({providers: [WireProvider.DISCORD, 7 as WireProvider, WireProvider.GOOGLE]})),
        })

        expect(await backend.signInOptions()).toEqual(["discord", "google"])
    })

    // An old server, or one with auth off, has no such route.
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

        expect(await backend.startSignIn("discord")).toBe("https://discord.example/authorize")
        expect(startSignIn).toHaveBeenCalledWith({provider: WireProvider.DISCORD})
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

    // A retried CompleteSignIn would spend a code that is good once.
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

    // Every call here is about the cookie's account, so every one carries it.
    it("asks for the sign-in options with credentials", async () => {
        const fetch = vi.fn(async (): Promise<Response> => {
            throw new ConnectError("no", Code.Unimplemented)
        })
        vi.stubGlobal("fetch", fetch)

        await new ConnectAccountBackend(newAuthServiceClient({baseUrl: "https://api.test"})).signInOptions()

        expect(String((fetch.mock.calls[0] as unknown[])[0])).toBe("https://api.test/auth.v1.AuthService/GetSignInOptions")
        expect(((fetch.mock.calls[0] as unknown[])[1] as RequestInit).credentials).toBe("include")
    })
})
