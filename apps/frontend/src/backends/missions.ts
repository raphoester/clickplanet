import {ChargeKind} from "../domain/bonus.ts"

export type MissionKind = "takeFor" | "takeFrom" | "takeIn"

export type MissionReward = {
    kind: ChargeKind
    amount: number
}

export type Mission = {
    id: string
    kind: MissionKind
    target: string
    goal: number
    progress: number
    paid: boolean
    reward: MissionReward
}

export type DayMissions = {
    countryCode: string
    enlisted: boolean
    endsAt: number
    missions: Mission[]
}

export interface MissionsBackend {
    missions(): Promise<DayMissions>
}
