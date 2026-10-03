import {afterEach, describe, expect, it, vi} from "vitest"
import {Code, ConnectError} from "@connectrpc/connect"
import {SubscriptionState as SubscriptionStatePb} from "../gen/grpc/marketing/v1/marketing_pb.ts"
import {MarketingError} from "./marketing.ts"
import {ConnectMarketingBackend} from "./marketingBackend.ts"
import {SESSION_HEADER, SessionProvider} from "./session.ts"

const refusing = (code: Code) => vi.fn(async () => {
    throw new ConnectError("no", code)
})

function sessionOf(...tokens: string[]): SessionProvider & {invalidate: ReturnType<typeof vi.fn>} {
    let minted = 0
    return {
        token: vi.fn(async () => tokens[Math.min(minted, tokens.length - 1)]),
        held: vi.fn(() => undefined),
        invalidate: vi.fn(() => {
            minted++
        }),
    }
}

function backendWith(methods: Record<string, unknown>, session: SessionProvider = sessionOf("token-1")) {
    return new ConnectMarketingBackend(methods as never, session)
}

const headersOf = (call: ReturnType<typeof vi.fn>, n = 0) =>
    (call.mock.calls[n] as unknown[])[1] as {headers?: Headers} | undefined

afterEach(() => vi.restoreAllMocks())

describe("ConnectMarketingBackend", () => {
    describe("offered", () => {
        it("asks without a token, so a guest mints nothing to learn it", async () => {
            const session = sessionOf("token-1")
            const getSubscription = refusing(Code.Unauthenticated)

            expect(await backendWith({getSubscription}, session).offered()).toBe(true)

            expect(session.token).not.toHaveBeenCalled()
            expect(headersOf(getSubscription)).toBeUndefined()
        })

        it("reads a server with no route as no season emails", async () => {
            for (const code of [Code.Unimplemented, Code.NotFound]) {
                expect(await backendWith({getSubscription: refusing(code)}).offered(), Code[code]).toBe(false)
            }
        })

        it("lets any other failure through", async () => {
            await expect(backendWith({getSubscription: refusing(Code.Internal)}).offered()).rejects.toBeInstanceOf(ConnectError)
        })
    })

    describe("subscription", () => {
        it("reads the state and the address with the click token", async () => {
            const getSubscription = vi.fn(async () => ({state: SubscriptionStatePb.WAITING, address: "ada@work.example"}))

            expect(await backendWith({getSubscription}).subscription()).toEqual({state: "waiting", address: "ada@work.example"})

            expect(headersOf(getSubscription)?.headers?.get(SESSION_HEADER)).toBe("token-1")
        })

        it("reads a state this build does not know as none", async () => {
            const getSubscription = vi.fn(async () => ({state: SubscriptionStatePb.UNSPECIFIED, address: ""}))

            expect(await backendWith({getSubscription}).subscription()).toEqual({state: "none", address: ""})
        })

        it("answers nothing when the server has no season emails", async () => {
            expect(await backendWith({getSubscription: refusing(Code.Unimplemented)}).subscription()).toBeUndefined()
        })

        it("mints again once when the token is refused", async () => {
            const session = sessionOf("token-1", "token-2")
            let calls = 0
            const getSubscription = vi.fn(async () => {
                if (calls++ === 0) throw new ConnectError("stale", Code.Unauthenticated)
                return {state: SubscriptionStatePb.ACTIVE, address: "ada@example.com"}
            })

            expect(await backendWith({getSubscription}, session).subscription()).toEqual({state: "active", address: "ada@example.com"})

            expect(session.invalidate).toHaveBeenCalledTimes(1)
            expect(headersOf(getSubscription, 1)?.headers?.get(SESSION_HEADER)).toBe("token-2")
        })
    })

    describe("subscribe", () => {
        it("sends the address and answers the state it took", async () => {
            const subscribe = vi.fn(async () => ({state: SubscriptionStatePb.ACTIVE}))

            expect(await backendWith({subscribe}).subscribe("ada@example.com")).toBe("active")

            expect((subscribe.mock.calls[0] as unknown[])[0]).toEqual({address: "ada@example.com"})
        })

        it("names each refusal", async () => {
            for (const [code, failure] of [
                [Code.InvalidArgument, "invalid"],
                [Code.PermissionDenied, "guest"],
                [Code.FailedPrecondition, "alreadySubscribed"],
                [Code.Unavailable, "unavailable"],
                [Code.ResourceExhausted, "tooManyTries"],
                [Code.Unimplemented, "off"],
                [Code.Internal, "failed"],
            ] as const) {
                const error = await backendWith({subscribe: refusing(code)}).subscribe("ada@example.com").catch((e) => e)

                expect(error, Code[code]).toBeInstanceOf(MarketingError)
                expect((error as MarketingError).failure, Code[code]).toBe(failure)
            }
        })

        it("does not send a write twice when the mailing list is down", async () => {
            const subscribe = refusing(Code.Unavailable)

            await backendWith({subscribe}).subscribe("ada@example.com").catch(() => {})

            expect(subscribe).toHaveBeenCalledTimes(1)
        })

        it("is notSignedIn when a fresh token is refused too", async () => {
            const session = sessionOf("token-1", "token-2")

            const error = await backendWith({subscribe: refusing(Code.Unauthenticated)}, session).subscribe("ada@example.com").catch((e) => e)

            expect((error as MarketingError).failure).toBe("notSignedIn")
            expect(session.invalidate).toHaveBeenCalledTimes(1)
        })
    })

    describe("unsubscribe", () => {
        it("carries the click token", async () => {
            const unsubscribe = vi.fn(async () => ({}))

            await backendWith({unsubscribe}).unsubscribe()

            expect(headersOf(unsubscribe)?.headers?.get(SESSION_HEADER)).toBe("token-1")
        })
    })
})
