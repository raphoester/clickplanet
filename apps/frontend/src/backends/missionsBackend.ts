import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {MissionService} from "../gen/grpc/missions/v1/missions_connect.ts"
import {Mission as MissionPb, MissionReward as MissionRewardPb} from "../gen/grpc/missions/v1/missions_pb.ts"
import {chargeKindOf} from "./bonusKind.ts"
import {DayMissions, Mission, MissionOrder, MissionReward, MissionsBackend} from "./missions.ts"
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
