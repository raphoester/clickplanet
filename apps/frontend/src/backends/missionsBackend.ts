import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {MissionService} from "../gen/grpc/missions/v1/missions_connect.ts"
import {
    FlagMission as FlagMissionPb,
    Mission as MissionPb,
    MissionReward as MissionRewardPb,
} from "../gen/grpc/missions/v1/missions_pb.ts"
import {chargeKindOf} from "./bonusKind.ts"
import {
    DayMissions,
    FlagMission,
    FlagMissionOrder,
    Mission,
    MissionOrder,
    MissionReward,
    MissionsBackend,
} from "./missions.ts"
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
            endsAt: Number(res.endsAtUnixMs),
            missions: res.missions.flatMap((mission) => missionOf(mission) ?? []),
            flagMission: res.flagMission && flagMissionOf(res.flagMission),
        }
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

function missionOf(mission: MissionPb): Mission | undefined {
    const order = orderOf(mission.kind)
    const reward = rewardOf(mission.reward)
    if (!order || !reward) return undefined
    return {id: mission.id, order, reward, done: mission.done, paid: mission.paid}
}

function orderOf(kind: MissionPb["kind"]): MissionOrder | undefined {
    switch (kind.case) {
        case "takeFor":
        case "takeFrom":
        case "takeIn":
            return {kind: kind.case, country: kind.value.countryId, goal: kind.value.goal, taken: kind.value.taken}
        case "takeFromIn":
            return {
                kind: kind.case, country: kind.value.countryId, ground: kind.value.groundId,
                goal: kind.value.goal, taken: kind.value.taken,
            }
        case "bombIn":
            return {kind: kind.case, country: kind.value.countryId, goal: kind.value.goal, reached: kind.value.dropped}
        case "shieldTiles":
            return {kind: kind.case, shields: kind.value.shields, goal: kind.value.goal, reached: kind.value.shielded}
        case "catchBoxes":
            return {kind: kind.case, goal: kind.value.goal, reached: kind.value.caught}
        case "answerQuizzes":
            return {kind: kind.case, goal: kind.value.goal, reached: kind.value.answered}
        case "fortifyTiles":
            return {kind: kind.case, goal: kind.value.goal, reached: kind.value.fortified}
        case undefined:
            return undefined
    }
}

function flagMissionOf(mission: FlagMissionPb): FlagMission | undefined {
    const order = flagOrderOf(mission.kind)
    return order && {id: mission.id, order, points: mission.points, done: mission.done}
}

function flagOrderOf(kind: FlagMissionPb["kind"]): FlagMissionOrder | undefined {
    switch (kind.case) {
        case "takeIn":
            return {kind: kind.case, country: kind.value.countryId, goal: kind.value.goal, taken: kind.value.taken}
        case "breakShields":
            return {kind: kind.case, country: kind.value.countryId, goal: kind.value.goal, reached: kind.value.broken}
        case undefined:
            return undefined
    }
}

function rewardOf(reward: MissionRewardPb | undefined): MissionReward | undefined {
    switch (reward?.kind.case) {
        case "charges": {
            const kind = chargeKindOf(reward.kind.value.kind)
            return kind && {kind, amount: reward.kind.value.amount}
        }
        case undefined:
            return undefined
    }
}
