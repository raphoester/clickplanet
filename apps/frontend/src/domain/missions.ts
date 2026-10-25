import {Mission, MissionKind, MissionReward} from "../backends/missions.ts"
import {BonusReward, BonusRules} from "./bonus.ts"

const PREPOSITIONS: Record<MissionKind, string> = {
    takeFor: "for",
    takeFrom: "from",
    takeIn: "in",
}

export function orderOf(mission: Mission): string {
    return `Take ${mission.goal} tiles ${PREPOSITIONS[mission.kind]}`
}

export function done(mission: Mission): boolean {
    return mission.progress >= mission.goal
}

export function collectable(missions: readonly Mission[]): number {
    return missions.filter((mission) => done(mission) && !mission.collected).length
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
