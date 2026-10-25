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
    collected: boolean
    reward: MissionReward
}

export type DayMissions = {
    countryCode: string
    enlisted: boolean
    endsAt: number
    missions: Mission[]
}

export type MissionFailure = "guest" | "full" | "failed"

export class MissionError extends Error {
    constructor(public readonly failure: MissionFailure, options?: {cause?: unknown}) {
        super(`mission refused: ${failure}`, options)
        this.name = "MissionError"
    }
}

export interface MissionsBackend {
    missions(): Promise<DayMissions>

    collectMission(id: string): Promise<MissionReward>
}
