import {afterEach, describe, expect, it, vi} from "vitest"
import {Code, ConnectError} from "@connectrpc/connect"
import {
    CompleteDiscordLinkResponse,
    GetDiscordLinkResponse,
    StartDiscordLinkResponse,
} from "../gen/grpc/alerts/v1/alerts_pb.ts"
import {AlertsFailure, NOT_OFFERED} from "./alerts.ts"
import {ConnectAlertsBackend} from "./alertsBackend.ts"
import {SESSION_HEADER, SessionProvider} from "./session.ts"

const refusing = (code: Code) => vi.fn(async () => {
    throw new ConnectError("no", code)
})

function sessionOf(...tokens: string[]): SessionProvider & {invalidate: ReturnType<typeof vi.fn>} {
    let minted = 0
    return {
        token: vi.fn(async () => tokens[Math.min(minted, tokens.length - 1)]),
        held: vi.fn(() => undefined), identity: vi.fn(async () => undefined), heldIdentity: vi.fn(() => undefined),
        invalidate: vi.fn(() => {
            minted++
        }),
    }
}

const backendWith = (methods: Record<string, unknown>, session: SessionProvider = sessionOf("token-1")) =>
    new ConnectAlertsBackend(methods as never, session)

const headersOf = (call: ReturnType<typeof vi.fn>, n = 0) =>
    ((call.mock.calls[n] as unknown[])[1] as {headers: Headers}).headers

afterEach(() => vi.restoreAllMocks())

describe("ConnectAlertsBackend", () => {
    it("reads whether the link is offered and made, with the click token", async () => {
        const getDiscordLink = vi.fn(async () => new GetDiscordLinkResponse({offered: true, linked: false, serverUrl: "https://discord.com/channels/1"}))

        expect(await backendWith({getDiscordLink}).discordLink()).toEqual({offered: true, linked: false})
        expect(headersOf(getDiscordLink).get(SESSION_HEADER)).toBe("token-1")
    })

    it("reads a server without the service as no offer", async () => {
        for (const code of [Code.Unimplemented, Code.NotFound]) {
            expect(await backendWith({getDiscordLink: refusing(code)}).discordLink()).toEqual(NOT_OFFERED)
        }
    })

    it("reports any other failure of the read as it is", async () => {
        await expect(backendWith({getDiscordLink: refusing(Code.Internal)}).discordLink())
            .rejects.toMatchObject({code: Code.Internal})
    })

    it("starts the link with the click token and answers Discord's page", async () => {
        const startDiscordLink = vi.fn(async () => new StartDiscordLinkResponse({authorizationUrl: "https://discord.com/oauth2/authorize?x"}))

        expect(await backendWith({startDiscordLink}).startDiscordLink()).toBe("https://discord.com/oauth2/authorize?x")
        expect(headersOf(startDiscordLink).get(SESSION_HEADER)).toBe("token-1")
    })

    it("trades the code once, with the click token, and answers the server's page", async () => {
        const completeDiscordLink = vi.fn(async () => new CompleteDiscordLinkResponse({serverUrl: "https://discord.com/channels/1"}))

        expect(await backendWith({completeDiscordLink}).completeDiscordLink("abc", "xyz")).toBe("https://discord.com/channels/1")
        expect(completeDiscordLink).toHaveBeenCalledTimes(1)
        expect(completeDiscordLink).toHaveBeenCalledWith({code: "abc", state: "xyz"}, expect.anything())
        expect(headersOf(completeDiscordLink).get(SESSION_HEADER)).toBe("token-1")
    })

    it("sends a call refused for its token once more, with a fresh session", async () => {
        const session = sessionOf("token-1", "token-2")
        const completeDiscordLink = vi.fn()
            .mockRejectedValueOnce(new ConnectError("no", Code.Unauthenticated))
            .mockResolvedValueOnce(new CompleteDiscordLinkResponse({serverUrl: "https://discord.com/channels/1"}))

        expect(await backendWith({completeDiscordLink}, session).completeDiscordLink("abc", "xyz")).toBe("https://discord.com/channels/1")
        expect(session.invalidate).toHaveBeenCalledTimes(1)
        expect(headersOf(completeDiscordLink, 1).get(SESSION_HEADER)).toBe("token-2")
    })

    it.each<[Code, AlertsFailure]>([
        [Code.FailedPrecondition, "startAgain"],
        [Code.PermissionDenied, "refused"],
        [Code.Unauthenticated, "notSignedIn"],
        [Code.ResourceExhausted, "tooManyTries"],
        [Code.Unimplemented, "off"],
        [Code.Internal, "failed"],
        [Code.InvalidArgument, "failed"],
    ])("maps a trade refused with %s to %s", async (code, failure) => {
        const completeDiscordLink = refusing(code)

        await expect(backendWith({completeDiscordLink}).completeDiscordLink("abc", "xyz"))
            .rejects.toMatchObject({failure})
    })

    it("maps a refused start the same way", async () => {
        await expect(backendWith({startDiscordLink: refusing(Code.ResourceExhausted)}).startDiscordLink())
            .rejects.toMatchObject({failure: "tooManyTries"})
    })

    it("maps a failed mint to failed", async () => {
        const session = sessionOf("token-1")
        session.token = vi.fn(async () => {
            throw new Error("mint refused")
        })

        await expect(backendWith({startDiscordLink: vi.fn()}, session).startDiscordLink())
            .rejects.toMatchObject({failure: "failed"})
    })
})
