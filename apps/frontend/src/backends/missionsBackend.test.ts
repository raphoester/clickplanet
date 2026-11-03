import {afterEach, describe, expect, it, vi} from "vitest"
import {Code, ConnectError} from "@connectrpc/connect"
import {
    FlagMission as FlagMissionPb,
    GetMissionsResponse as GetMissionsResponsePb,
    Mission as MissionPb,
    MissionEvent as MissionEventPb,
} from "../gen/grpc/missions/v1/missions_pb.ts"
import {BonusKind as BonusKindPb} from "../gen/grpc/planet/v1/planet_pb.ts"
import {DayMissions} from "./missions.ts"
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

const never = () => new Promise<never>(() => {})

const refusing = (code: Code) => vi.fn((): AsyncIterable<MissionEventPb> => ({
    [Symbol.asyncIterator]: () => ({
        next: async () => {
            throw new ConnectError("refused", code)
        },
    }),
}))

describe("ConnectMissionsBackend.follow", () => {
    it("hands on each day the stream sends with the click token, and skips the heartbeats", async () => {
        const listenForEvents = vi.fn(async function* () {
            yield new MissionEventPb({event: {case: "day", value: new GetMissionsResponsePb({endsAtUnixMs: 1_000n})}})
            yield new MissionEventPb({event: {case: "heartbeat", value: {}}})
            yield new MissionEventPb({event: {case: "day", value: new GetMissionsResponsePb({endsAtUnixMs: 2_000n, missions: [
                new MissionPb({
                    id: "m-1", done: true, paid: true, kind: {case: "takeFor", value: {countryId: "fr", goal: 60, taken: 60}},
                    reward: {kind: {case: "charges", value: {kind: BonusKindPb.BOMB, amount: 1}}},
                }),
            ]})}})
            await never()
        })
        const days: DayMissions[] = []

        const stop = backendWith({listenForEvents}).follow((day) => days.push(day))
        await vi.waitFor(() => expect(days).toHaveLength(2))
        stop()

        expect((listenForEvents.mock.calls[0] as unknown[])[1]).toMatchObject({headers: {[SESSION_HEADER]: "token-1"}})
        expect(days).toEqual([
            {endsAt: 1_000, missions: []},
            {endsAt: 2_000, missions: [{
                id: "m-1", order: {kind: "takeFor", country: "fr", goal: 60, taken: 60}, done: true, paid: true,
                reward: {kind: "bomb", amount: 1},
            }]},
        ])
    })

    it("does not reconnect to a server without the stream", async () => {
        vi.useFakeTimers()
        for (const code of [Code.Unimplemented, Code.NotFound]) {
            const listenForEvents = refusing(code)

            backendWith({listenForEvents}).follow(() => {})
            await vi.advanceTimersByTimeAsync(60_000)

            expect(listenForEvents).toHaveBeenCalledTimes(1)
        }
        vi.useRealTimers()
    })

    it("drops a token the stream refuses, so the next try mints another", async () => {
        vi.useFakeTimers()
        const provider = session()
        const stop = new ConnectMissionsBackend({listenForEvents: refusing(Code.Unauthenticated)} as never, provider)
            .follow(() => {})

        await vi.advanceTimersByTimeAsync(0)
        stop()
        vi.useRealTimers()

        expect(provider.invalidate).toHaveBeenCalled()
    })
})

describe("ConnectMissionsBackend", () => {
    it("reads the day's missions with the click token, and skips a kind or a reward it does not know", async () => {
        const getMissions = vi.fn(async () => new GetMissionsResponsePb({
            endsAtUnixMs: 1_000n,
            missions: [
                new MissionPb({
                    id: "m-2", done: true, paid: true,
                    kind: {case: "takeFrom", value: {countryId: "dz", goal: 40, taken: 40}},
                    reward: {kind: {case: "charges", value: {kind: BonusKindPb.SPREAD_CLICKS, amount: 4}}},
                }),
                new MissionPb({id: "m-4", reward: {kind: {case: "charges", value: {kind: BonusKindPb.SHIELDS, amount: 1}}}}),
                new MissionPb({id: "m-5", kind: {case: "takeFor", value: {countryId: "fr", goal: 60}}}),
            ],
        }))

        const day = await backendWith({getMissions}).missions()

        expect((getMissions.mock.calls[0] as unknown[])[0]).toEqual({})
        expect(headersOf(getMissions).get(SESSION_HEADER)).toBe("token-1")
        expect(day).toEqual({
            endsAt: 1_000,
            missions: [{
                id: "m-2", order: {kind: "takeFrom", country: "dz", goal: 40, taken: 40}, done: true, paid: true,
                reward: {kind: "spreadClicks", amount: 4},
            }],
        })
    })

    it("reads the flag's mission, and skips one whose kind it does not know", async () => {
        const getMissions = vi.fn()
            .mockResolvedValueOnce(new GetMissionsResponsePb({flagMission: new FlagMissionPb({
                id: "f-1", points: 3, done: true, kind: {case: "breakShields", value: {countryId: "dz", goal: 200, broken: 200}},
            })}))
            .mockResolvedValueOnce(new GetMissionsResponsePb({flagMission: new FlagMissionPb({id: "f-2", points: 1})}))
        const backend = backendWith({getMissions})

        expect((await backend.missions()).flagMission).toEqual({
            id: "f-1", points: 3, done: true, order: {kind: "breakShields", country: "dz", goal: 200, reached: 200},
        })
        expect((await backend.missions()).flagMission).toBeUndefined()
    })

    it("reads each kind of mission in its own shape", async () => {
        const reward = {kind: {case: "charges" as const, value: {kind: BonusKindPb.BOMB, amount: 1}}}
        const getMissions = vi.fn(async () => new GetMissionsResponsePb({
            missions: [
                new MissionPb({id: "raid", reward, kind: {case: "takeFromIn", value: {countryId: "dz", groundId: "br", goal: 60, taken: 7}}}),
                new MissionPb({id: "bomb", reward, kind: {case: "bombIn", value: {countryId: "br", goal: 2, dropped: 1}}}),
                new MissionPb({id: "walls", reward, kind: {case: "shieldTiles", value: {shields: 10, goal: 2, shielded: 1}}}),
                new MissionPb({id: "boxes", reward, kind: {case: "catchBoxes", value: {goal: 4, caught: 3}}}),
                new MissionPb({id: "quizzes", reward, kind: {case: "answerQuizzes", value: {goal: 3, answered: 2}}}),
                new MissionPb({id: "fortify", reward, kind: {case: "fortifyTiles", value: {goal: 40, fortified: 9}}}),
            ],
        }))

        const day = await backendWith({getMissions}).missions()

        expect(day.missions.map((mission) => mission.order)).toEqual([
            {kind: "takeFromIn", country: "dz", ground: "br", goal: 60, taken: 7},
            {kind: "bombIn", country: "br", goal: 2, reached: 1},
            {kind: "shieldTiles", shields: 10, goal: 2, reached: 1},
            {kind: "catchBoxes", goal: 4, reached: 3},
            {kind: "answerQuizzes", goal: 3, reached: 2},
            {kind: "fortifyTiles", goal: 40, reached: 9},
        ])
    })

    it("tries once more with a fresh session when the token was refused", async () => {
        const getMissions = vi.fn()
            .mockRejectedValueOnce(new ConnectError("expired", Code.Unauthenticated))
            .mockResolvedValueOnce(new GetMissionsResponsePb({endsAtUnixMs: 1_000n}))

        const day = await backendWith({getMissions}).missions()

        expect(day.endsAt).toBe(1_000)
        expect(getMissions).toHaveBeenCalledTimes(2)
    })
})
