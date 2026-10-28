import {ChargeKind} from "../domain/bonus.ts"

export type MissionReward = {
    kind: ChargeKind
    amount: number
}

export type TileOrder = {
    kind: "takeFor" | "takeFrom" | "takeIn"
    country: string
    goal: number
    taken: number
}

export type ShieldOrder = {
    kind: "breakShields"
    country: string
    goal: number
    broken: number
}

// One shape per kind of mission; a kind the client does not know is not read.
export type MissionOrder = TileOrder

export type FlagMissionOrder = TileOrder | ShieldOrder

export type Mission = {
    id: string
    order: MissionOrder
    reward: MissionReward
    done: boolean
    paid: boolean
}

// The flag's own: every player of it counts on it, and it gives the flag season points.
export type FlagMission = {
    id: string
    order: FlagMissionOrder
    points: number
    done: boolean
}

export type DayMissions = {
    endsAt: number
    missions: Mission[]
    flagMission?: FlagMission
}

export interface MissionsBackend {
    missions(): Promise<DayMissions>
}
