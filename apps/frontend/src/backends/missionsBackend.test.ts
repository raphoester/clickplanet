import {afterEach, describe, expect, it, vi} from "vitest"
import {Code, ConnectError} from "@connectrpc/connect"
import {
    CollectMissionResponse as CollectMissionResponsePb,
    GetMissionsResponse as GetMissionsResponsePb,
    Mission as MissionPb,
    MissionKind as MissionKindPb,
} from "../gen/grpc/missions/v1/missions_pb.ts"
import {BonusKind as BonusKindPb} from "../gen/grpc/planet/v1/planet_pb.ts"
import {MissionError} from "./missions.ts"
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
                    slot: 2, kind: MissionKindPb.TAKE_FROM, targetCountryId: "dz", goal: 40, progress: 12,
                    rewardKind: BonusKindPb.SPREAD_CLICKS, rewardAmount: 4,
                }),
                new MissionPb({slot: 4, kind: MissionKindPb.UNSPECIFIED, rewardKind: BonusKindPb.SHIELDS}),
            ],
        }))

        const day = await backendWith({getMissions}).missions()

        expect((getMissions.mock.calls[0] as unknown[])[0]).toEqual({})
        expect(headersOf(getMissions).get(SESSION_HEADER)).toBe("token-1")
        expect(day).toEqual({
            countryCode: "fr", enlisted: true, endsAt: 1_000,
            missions: [{
                slot: 2, kind: "takeFrom", target: "dz", goal: 40, progress: 12, collected: false,
                reward: {kind: "spreadClicks", amount: 4},
            }],
        })
    })

    it("answers what a collected mission paid", async () => {
        const collectMission = vi.fn(async () => new CollectMissionResponsePb({kind: BonusKindPb.BOMB, amount: 1}))

        expect(await backendWith({collectMission}).collectMission(3)).toEqual({kind: "bomb", amount: 1})
        expect((collectMission.mock.calls[0] as unknown[])[0]).toEqual({slot: 3})
    })

    it("reads a guest refused, and a reward that did not fit", async () => {
        for (const [code, failure] of [[Code.PermissionDenied, "guest"], [Code.FailedPrecondition, "full"], [Code.Internal, "failed"]] as const) {
            const refusing = vi.fn(async () => {
                throw new ConnectError("no", code)
            })

            const failed = await backendWith({collectMission: refusing}).collectMission(1).catch((e: unknown) => e)

            expect(failed).toBeInstanceOf(MissionError)
            expect((failed as MissionError).failure).toBe(failure)
        }
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
