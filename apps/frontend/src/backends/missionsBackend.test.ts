import {afterEach, describe, expect, it, vi} from "vitest"
import {Code, ConnectError} from "@connectrpc/connect"
import {
    GetMissionsResponse as GetMissionsResponsePb,
    Mission as MissionPb,
    MissionKind as MissionKindPb,
} from "../gen/grpc/missions/v1/missions_pb.ts"
import {BonusKind as BonusKindPb} from "../gen/grpc/planet/v1/planet_pb.ts"
import {ConnectMissionsBackend} from "./missionsBackend.ts"
import {SESSION_HEADER, SessionProvider} from "./session.ts"

const session = (): SessionProvider => ({
    token: vi.fn(async () => "token-1"),
    held: vi.fn(() => undefined), identity: vi.fn(async () => undefined), heldIdentity: vi.fn(() => undefined),
    invalidate: vi.fn(),
})

const backendWith = (methods: Record<string, unknown>) => new ConnectMissionsBackend(methods as never, session())

const headersOf = (call: ReturnType<typeof vi.fn>) => ((call.mock.calls[0] as unknown[])[1] as {headers: Headers}).headers

afterEach(() => vi.restoreAllMocks())

describe("ConnectMissionsBackend", () => {
    it("reads the day's missions with the click token", async () => {
        const getMissions = vi.fn(async () => new GetMissionsResponsePb({
            countryId: "fr", enlisted: true, endsAtUnixMs: 1_000n,
            missions: [
                new MissionPb({
                    id: "m-2", kind: MissionKindPb.TAKE_FROM, targetCountryId: "dz", goal: 40, progress: 40, paid: true,
                    rewardKind: BonusKindPb.SPREAD_CLICKS, rewardAmount: 4,
                }),
                new MissionPb({id: "m-4", kind: MissionKindPb.UNSPECIFIED, rewardKind: BonusKindPb.SHIELDS}),
            ],
        }))

        const day = await backendWith({getMissions}).missions()

        expect((getMissions.mock.calls[0] as unknown[])[0]).toEqual({})
        expect(headersOf(getMissions).get(SESSION_HEADER)).toBe("token-1")
        expect(day).toEqual({
            countryCode: "fr", enlisted: true, endsAt: 1_000,
            missions: [{
                id: "m-2", kind: "takeFrom", target: "dz", goal: 40, progress: 40, paid: true,
                reward: {kind: "spreadClicks", amount: 4},
            }],
        })
    })

    it("tries once more with a fresh session when the token was refused", async () => {
        const getMissions = vi.fn()
            .mockRejectedValueOnce(new ConnectError("expired", Code.Unauthenticated))
            .mockResolvedValueOnce(new GetMissionsResponsePb({countryId: "fr"}))

        const day = await backendWith({getMissions}).missions()

        expect(day.countryCode).toBe("fr")
        expect(getMissions).toHaveBeenCalledTimes(2)
    })
})
