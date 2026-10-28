import {Mission, MissionReward, ShieldOrder, TileOrder} from "../backends/missions.ts"
import {BonusReward, BonusRules} from "./bonus.ts"

const PREPOSITIONS: Record<TileOrder["kind"], string> = {
    takeFor: "for",
    takeFrom: "from",
    takeIn: "in",
}

export function orderOf(order: TileOrder): string {
    return `Take ${order.goal} tiles ${PREPOSITIONS[order.kind]}`
}

export function shieldOrderOf(order: ShieldOrder): string {
    return `Break ${order.goal} shields of`
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
