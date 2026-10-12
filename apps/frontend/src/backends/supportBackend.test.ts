import {describe, expect, it, vi} from "vitest"
import {Code, ConnectError, PromiseClient} from "@connectrpc/connect"
import {SupportService} from "../gen/grpc/support/v1/support_connect.ts"
import {SESSION_HEADER, SessionProvider} from "./session.ts"
import {SupportNoSessionError, SupportRateLimitedError, SupportRejectedError} from "./support.ts"
import {ConnectSupportBackend} from "./supportBackend.ts"

const session = (...tokens: string[]): SessionProvider => {
    const token = vi.fn(async () => tokens.length > 1 ? tokens.shift() : tokens[0])
    return {token, held: vi.fn(), identity: vi.fn(), heldIdentity: vi.fn(), invalidate: vi.fn()}
}

const noMint = (): SessionProvider => ({
    token: vi.fn(async () => {
        throw new Error("no mint")
    }),
    held: vi.fn(), identity: vi.fn(), heldIdentity: vi.fn(), invalidate: vi.fn(),
})

const clientOf = (suggestFeature: unknown) => ({suggestFeature}) as unknown as PromiseClient<typeof SupportService>

const phone = {
    build: "play-a1b2c3.js",
    viewportWidth: 390,
    viewportHeight: 844,
    pixelRatio: 3,
    connectionDown: true,
    gpuVendor: "Apple Inc.",
    gpuRenderer: "Apple GPU",
    webglVersion: "WebGL 2.0",
    maxTextureSize: 4096,
    contextLost: false,
    cores: 6,
    memoryGb: 4,
}

const headersOf = (call: ReturnType<typeof vi.fn>, n = 0) => (call.mock.calls[n][1] as {headers: Headers}).headers

describe("ConnectSupportBackend.suggestFeature", () => {
    it("sends the text and what the page knows, with the click token", async () => {
        const suggestFeature = vi.fn().mockResolvedValue({})

        await new ConnectSupportBackend(clientOf(suggestFeature), session("token-1")).suggestFeature("a night mode", phone)

        expect(suggestFeature.mock.calls[0][0]).toEqual({text: "a night mode", context: phone})
        expect(headersOf(suggestFeature).get(SESSION_HEADER)).toBe("token-1")
    })

    it("mints again and sends once more when the token is refused", async () => {
        const suggestFeature = vi.fn()
            .mockRejectedValueOnce(new ConnectError("no token", Code.Unauthenticated))
            .mockResolvedValue({})
        const provider = session("stale", "fresh")

        await new ConnectSupportBackend(clientOf(suggestFeature), provider).suggestFeature("a night mode", phone)

        expect(provider.invalidate).toHaveBeenCalledTimes(1)
        expect(suggestFeature).toHaveBeenCalledTimes(2)
        expect(headersOf(suggestFeature, 1).get(SESSION_HEADER)).toBe("fresh")
    })

    it("gives up on a second refusal, with no third call", async () => {
        const suggestFeature = vi.fn().mockRejectedValue(new ConnectError("no token", Code.Unauthenticated))

        await expect(new ConnectSupportBackend(clientOf(suggestFeature), session("token-1")).suggestFeature("a night mode", phone))
            .rejects.toBeInstanceOf(SupportNoSessionError)
        expect(suggestFeature).toHaveBeenCalledTimes(2)
    })

    it("sends nothing when no token can be minted", async () => {
        const suggestFeature = vi.fn()

        await expect(new ConnectSupportBackend(clientOf(suggestFeature), noMint()).suggestFeature("a night mode", phone))
            .rejects.toBeInstanceOf(SupportNoSessionError)
        expect(suggestFeature).not.toHaveBeenCalled()
    })

    it.each([
        [Code.ResourceExhausted, SupportRateLimitedError],
        [Code.InvalidArgument, SupportRejectedError],
    ])("reads the server's refusal %s as its own error", async (code, expected) => {
        const suggestFeature = vi.fn().mockRejectedValue(new ConnectError("refused", code))

        await expect(new ConnectSupportBackend(clientOf(suggestFeature), session("token-1")).suggestFeature("a night mode", phone))
            .rejects.toBeInstanceOf(expected)
        expect(suggestFeature).toHaveBeenCalledTimes(1)
    })

    it("does not send again after a failure that may have landed", async () => {
        const dropped = new ConnectError("the connection dropped", Code.Unavailable)
        const suggestFeature = vi.fn().mockRejectedValue(dropped)

        await expect(new ConnectSupportBackend(clientOf(suggestFeature), session("token-1")).suggestFeature("a night mode", phone))
            .rejects.toBe(dropped)
        expect(suggestFeature).toHaveBeenCalledTimes(1)
    })
})

describe("ConnectSupportBackend.reportBug", () => {
    const backendOf = (reportBug: unknown, provider: SessionProvider) =>
        new ConnectSupportBackend({reportBug} as unknown as PromiseClient<typeof SupportService>, provider)

    it("sends the text and what the page knows, with the click token", async () => {
        const reportBug = vi.fn().mockResolvedValue({})

        await backendOf(reportBug, session("token-1")).reportBug("the globe is black", phone)

        expect(reportBug.mock.calls[0][0]).toEqual({text: "the globe is black", context: phone})
        expect(headersOf(reportBug).get(SESSION_HEADER)).toBe("token-1")
    })

    it("mints again once when the token is refused, and reads the refusals as a suggestion's", async () => {
        const refusedOnce = vi.fn()
            .mockRejectedValueOnce(new ConnectError("no token", Code.Unauthenticated))
            .mockResolvedValue({})
        const provider = session("stale", "fresh")

        await backendOf(refusedOnce, provider).reportBug("the globe is black", phone)

        expect(provider.invalidate).toHaveBeenCalledTimes(1)
        expect(headersOf(refusedOnce, 1).get(SESSION_HEADER)).toBe("fresh")

        const limited = vi.fn().mockRejectedValue(new ConnectError("too many", Code.ResourceExhausted))
        await expect(backendOf(limited, session("token-1")).reportBug("the globe is black", phone))
            .rejects.toBeInstanceOf(SupportRateLimitedError)
        expect(limited).toHaveBeenCalledTimes(1)
    })
})
