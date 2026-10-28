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

// One shape per kind of mission; a kind the client does not know is not read.
export type MissionOrder = TileOrder

export type Mission = {
    id: string
    order: MissionOrder
    reward: MissionReward
    done: boolean
    paid: boolean
}

export type DayMissions = {
    endsAt: number
    missions: Mission[]
}

export interface MissionsBackend {
    missions(): Promise<DayMissions>
}
