import {DayMissions, FlagMission, FlagMissionOrder, Mission, MissionOrder, MissionReward} from "../backends/missions.ts"
import {BonusReward, BonusRules} from "./bonus.ts"

// What an order asks, up to the country named after it, if any.
export function orderOf(order: MissionOrder | FlagMissionOrder): string {
    switch (order.kind) {
        case "takeFor":
            return `Take ${order.goal} tiles for`
        case "takeFrom":
        case "takeFromIn":
            return `Take ${order.goal} tiles from`
        case "takeIn":
            return `Take ${order.goal} tiles in`
        case "bombIn":
            return `Drop ${counted(order.goal, "a bomb", "bombs")} on`
        case "shieldTiles":
            return `Put ${order.shields} shields on ${counted(order.goal, "a tile", "tiles")}`
        case "catchBoxes":
            return `Catch ${counted(order.goal, "a box", "boxes")}`
        case "answerQuizzes":
            return `Answer ${counted(order.goal, "a quiz", "quizzes")} right`
        case "fortifyTiles":
            return `Fortify ${counted(order.goal, "a tile", "tiles")}`
        case "breakShields":
            return `Break ${order.goal} shields of`
    }
}

function counted(count: number, one: string, many: string): string {
    return count === 1 ? one : `${count} ${many}`
}

export function reachedOf(order: MissionOrder | FlagMissionOrder): number {
    switch (order.kind) {
        case "takeFor":
        case "takeFrom":
        case "takeIn":
        case "takeFromIn":
            return Math.min(order.taken, order.goal)
        case "bombIn":
        case "shieldTiles":
        case "catchBoxes":
        case "answerQuizzes":
        case "fortifyTiles":
        case "breakShields":
            return Math.min(order.reached, order.goal)
    }
}

export const SEEN_MISSIONS_STORAGE_KEY = "clickplanet-missions-seen"

const STAGES = ["new", "done", "paid"] as const

// A mission is news again each time it moves on: drawn, then done, then paid.
export type MissionStage = typeof STAGES[number]

export type SeenMissions = Readonly<Record<string, MissionStage>>

export function stageOf(mission: Mission): MissionStage {
    if (mission.paid) return "paid"
    return mission.done ? "done" : "new"
}

export function unseen(missions: readonly Mission[], seen: SeenMissions): number {
    return missions.filter((mission) => {
        const was = seen[mission.id]
        return was === undefined || STAGES.indexOf(was) < STAGES.indexOf(stageOf(mission))
    }).length
}

// Only the missions shown are kept: another day's ids would never be read again.
export function seenNow(missions: readonly Mission[]): SeenMissions {
    return Object.fromEntries(missions.map((mission) => [mission.id, stageOf(mission)]))
}

export function parseSeenMissions(raw: string | null): SeenMissions {
    if (raw === null) return {}

    let stored: unknown
    try {
        stored = JSON.parse(raw)
    } catch {
        return {}
    }
    if (typeof stored !== "object" || stored === null || Array.isArray(stored)) return {}

    return Object.fromEntries(Object.entries(stored)
        .filter((entry): entry is [string, MissionStage] => STAGES.includes(entry[1] as MissionStage)))
}

export function newlyPaid(before: readonly Mission[], after: readonly Mission[]): Mission[] {
    return after.filter((mission) => mission.paid && before.some((was) => was.id === mission.id && !was.paid))
}

// The flag's mission, when it was read not done and is done now.
export function newlyDoneFlagMission(before: DayMissions, after: DayMissions): FlagMission | undefined {
    const was = before.flagMission
    const now = after.flagMission
    if (!was || !now || was.id !== now.id || was.done || !now.done) return undefined
    return now
}

export function awardOf(reward: MissionReward, rules: Pick<BonusRules, "blastRadius" | "enclosureMaxTiles">): BonusReward {
    switch (reward.kind) {
        case "refill":
            return {kind: "refill"}
        case "spreadClicks":
            return {kind: "spreadClicks", clicks: reward.amount}
        case "bomb":
            return {kind: "bomb", radius: rules.blastRadius}
        case "encloseClicks":
            return {kind: "encloseClicks", shapes: reward.amount, maxTiles: rules.enclosureMaxTiles}
        case "shields":
            return {kind: "shields", shields: reward.amount}
    }
}

const MINUTE_MS = 60_000
const HOUR_MS = 60 * MINUTE_MS

export function timeLeft(endsAt: number, now: number): string {
    const left = Math.max(0, endsAt - now)
    const hours = Math.floor(left / HOUR_MS)
    const minutes = Math.ceil(left % HOUR_MS / MINUTE_MS)
    if (minutes === 60) return `${hours + 1}h 00m`
    return hours > 0 ? `${hours}h ${String(minutes).padStart(2, "0")}m` : `${minutes}m`
}
