import {DayMissions, FlagMission, FlagMissionOrder, Mission, MissionOrder, MissionReward} from "../backends/missions.ts"
import {BonusReward, BonusRules} from "./bonus.ts"
import {Countdown, countdownTo} from "./seasonClock.ts"

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

export function timeLeft(endsAt: number, now: number): Countdown {
    return countdownTo(Math.max(endsAt, now), now)
}
