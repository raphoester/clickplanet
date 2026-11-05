import {afterEach, describe, expect, it, vi} from "vitest"
import {Code, ConnectError} from "@connectrpc/connect"
import {GetDiscordLinkResponse} from "../gen/grpc/alerts/v1/alerts_pb.ts"
import {NOT_OFFERED} from "./alerts.ts"
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

        expect(await backendWith({getDiscordLink}).discordLink())
            .toEqual({offered: true, linked: false, serverUrl: "https://discord.com/channels/1"})
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

    it("sends a read refused for its token once more, with a fresh session", async () => {
        const session = sessionOf("token-1", "token-2")
        const getDiscordLink = vi.fn()
            .mockRejectedValueOnce(new ConnectError("no", Code.Unauthenticated))
            .mockResolvedValueOnce(new GetDiscordLinkResponse({offered: true, linked: true}))

        expect(await backendWith({getDiscordLink}, session).discordLink()).toMatchObject({linked: true})
        expect(session.invalidate).toHaveBeenCalledTimes(1)
        expect(headersOf(getDiscordLink, 1).get(SESSION_HEADER)).toBe("token-2")
    })
})
