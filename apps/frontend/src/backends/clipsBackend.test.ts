import {afterEach, describe, expect, it, vi} from "vitest"
import {Timestamp} from "@bufbuild/protobuf"
import {Code, ConnectError} from "@connectrpc/connect"
import {
    Appearance as AppearancePb,
    ClipLink as ClipLinkPb,
    GetAppearancesResponse as GetAppearancesResponsePb,
} from "../gen/grpc/clips/v1/appearances_pb.ts"
import {ConnectClipsBackend} from "./clipsBackend.ts"
import {SESSION_HEADER, SessionProvider} from "./session.ts"

const sessionWith = (identity: string | undefined): SessionProvider => ({
    token: vi.fn(async () => "click-token"),
    held: vi.fn(() => undefined), identity: vi.fn(async () => identity), heldIdentity: vi.fn(() => identity),
    invalidate: vi.fn(),
})

const backendWith = (methods: Record<string, unknown>, session: SessionProvider = sessionWith("identity-1")) =>
    new ConnectClipsBackend(methods as never, session)

const headersOf = (call: ReturnType<typeof vi.fn>) => ((call.mock.calls[0] as unknown[])[1] as {headers: Headers}).headers

afterEach(() => vi.restoreAllMocks())

describe("ConnectClipsBackend", () => {
    it("reads the clips that name the player with the identity token, and skips a platform it does not know", async () => {
        const getAppearances = vi.fn(async () => new GetAppearancesResponsePb({
            appearances: [new AppearancePb({
                postId: "post-1",
                title: "FRANCE VS TURKEY",
                scheduledAt: Timestamp.fromDate(new Date(1_700_000_000_000)),
                links: [
                    new ClipLinkPb({platform: "tiktok", url: ""}),
                    new ClipLinkPb({platform: "instagram", url: "https://www.instagram.com/reel/1"}),
                    new ClipLinkPb({platform: "youtube", url: "https://www.youtube.com/shorts/abc"}),
                ],
            })],
        }))

        const appearances = await backendWith({getAppearances}).appearances()

        expect(headersOf(getAppearances).get(SESSION_HEADER)).toBe("identity-1")
        expect(appearances).toEqual([{
            postId: "post-1",
            title: "FRANCE VS TURKEY",
            scheduledAt: 1_700_000_000_000,
            links: [{platform: "tiktok", url: ""}, {platform: "youtube", url: "https://www.youtube.com/shorts/abc"}],
        }])
    })

    it("reads nothing, and asks nothing, without an identity", async () => {
        const getAppearances = vi.fn()

        expect(await backendWith({getAppearances}, sessionWith(undefined)).appearances()).toEqual([])
        expect(getAppearances).not.toHaveBeenCalled()
    })

    it("reads a server without clips, or a token refused, as no clip", async () => {
        for (const code of [Code.Unimplemented, Code.NotFound, Code.Unauthenticated]) {
            const getAppearances = vi.fn().mockRejectedValue(new ConnectError("no", code))

            expect(await backendWith({getAppearances}).appearances()).toEqual([])
        }
    })

    it("reports any other failure as it is", async () => {
        const getAppearances = vi.fn().mockRejectedValue(new ConnectError("no", Code.Internal))

        await expect(backendWith({getAppearances}).appearances()).rejects.toMatchObject({code: Code.Internal})
    })
})
