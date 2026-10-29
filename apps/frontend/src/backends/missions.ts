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

export type RaidOrder = {
    kind: "takeFromIn"
    country: string
    ground: string
    goal: number
    taken: number
}

export type BombOrder = {
    kind: "bombIn"
    country: string
    goal: number
    reached: number
}

export type ShieldOrder = {
    kind: "shieldTiles"
    shields: number
    goal: number
    reached: number
}

export type CountOrder = {
    kind: "catchBoxes" | "answerQuizzes" | "fortifyTiles"
    goal: number
    reached: number
}

// One shape per kind of mission; a kind the client does not know is not read.
export type MissionOrder = TileOrder | RaidOrder | BombOrder | ShieldOrder | CountOrder

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
