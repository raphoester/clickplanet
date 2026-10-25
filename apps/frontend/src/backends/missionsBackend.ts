import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {MissionService} from "../gen/grpc/missions/v1/missions_connect.ts"
import {Mission as MissionPb, MissionKind as MissionKindPb} from "../gen/grpc/missions/v1/missions_pb.ts"
import {chargeKindOf} from "./bonusKind.ts"
import {DayMissions, Mission, MissionError, MissionFailure, MissionKind, MissionReward, MissionsBackend} from "./missions.ts"
import {SESSION_HEADER, SessionProvider} from "./session.ts"
import {Config, retrying} from "./transport.ts"

export function newMissionServiceClient(config: Config): PromiseClient<typeof MissionService> {
    return createPromiseClient(MissionService, createConnectTransport({
        baseUrl: config.baseUrl,
        interceptors: config.interceptors,
        useBinaryFormat: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}

export class ConnectMissionsBackend implements MissionsBackend {
    constructor(
        private readonly client: PromiseClient<typeof MissionService>,
        private readonly session: SessionProvider,
    ) {
    }

    public async missions(): Promise<DayMissions> {
        const res = await this.authenticated((headers) =>
            retrying(() => this.client.getMissions({}, {headers}), "GetMissions"))
        return {
            countryCode: res.countryId,
            enlisted: res.enlisted,
            endsAt: Number(res.endsAtUnixMs),
            missions: res.missions.flatMap((mission) => missionOf(mission) ?? []),
        }
    }

    public async collectMission(slot: number): Promise<MissionReward> {
        let res
        try {
            res = await this.authenticated((headers) => this.client.collectMission({slot}, {headers}))
        } catch (e) {
            throw new MissionError(missionFailureOf(e), {cause: e})
        }
        const kind = chargeKindOf(res.kind)
        if (!kind) throw new MissionError("failed")
        return {kind, amount: res.amount}
    }

    private async authenticated<T>(call: (headers: Headers) => Promise<T>): Promise<T> {
        try {
            return await call(await this.headers())
        } catch (e) {
            if (!(e instanceof ConnectError) || e.code !== Code.Unauthenticated) throw e

            this.session.invalidate()
            return await call(await this.headers())
        }
    }

    private async headers(): Promise<Headers> {
        const token = await this.session.token()

        const headers = new Headers()
        if (token) headers.set(SESSION_HEADER, token)
        return headers
    }
}

const MISSION_KINDS: Partial<Record<MissionKindPb, MissionKind>> = {
    [MissionKindPb.TAKE_FOR]: "takeFor",
    [MissionKindPb.TAKE_FROM]: "takeFrom",
    [MissionKindPb.TAKE_IN]: "takeIn",
}

function missionOf(mission: MissionPb): Mission | undefined {
    const kind = MISSION_KINDS[mission.kind]
    const reward = chargeKindOf(mission.rewardKind)
    if (!kind || !reward) return undefined
    return {
        slot: mission.slot,
        kind,
        target: mission.targetCountryId,
        goal: mission.goal,
        progress: mission.progress,
        collected: mission.collected,
        reward: {kind: reward, amount: mission.rewardAmount},
    }
}

const MISSION_FAILURES: Partial<Record<Code, MissionFailure>> = {
    [Code.PermissionDenied]: "guest",
    [Code.FailedPrecondition]: "full",
}

function missionFailureOf(e: unknown): MissionFailure {
    return (e instanceof ConnectError && MISSION_FAILURES[e.code]) || "failed"
}
