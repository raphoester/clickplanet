import {Duration} from "@bufbuild/protobuf"
import {LONG_CALL_MS} from "../api.ts"
import {PlanetAdmin} from "../clients.ts"
import {GetRulesResponse, RuleChangeSource, RuleSet} from "../gen/grpc/planet/v1/admin_pb.ts"

export const DEFAULT_SET = "default"

// The set the newest change put in force; with no change, the config's own.
export function inForceOf(rules: GetRulesResponse): string {
    return rules.changes[0]?.ruleSet ?? DEFAULT_SET
}

export function tempoOf(set: RuleSet): string {
    if (set.frozen) return "Map frozen"
    return set.tempo === 1 ? "As configured" : `${set.tempo}× as fast`
}

export function boxesOf(set: RuleSet): string {
    const boxes = set.boxes
    if (boxes === undefined || set.frozen) return "—"

    const shortest = spanOf(boxes.minInterval)
    const longest = spanOf(boxes.maxInterval)
    const every = shortest === longest ? shortest : `${shortest} to ${longest}`
    return `Every ${every}, ${boxes.maxChargesPerHour} an hour at most`
}

function spanOf(duration: Duration | undefined): string {
    const seconds = Number(duration?.seconds ?? 0n)
    return seconds % 60 === 0 ? `${seconds / 60} min` : `${seconds} s`
}

export function crowdOf(set: RuleSet): string {
    const crowd = set.antiBot?.crowd ?? 0
    return crowd > 0 ? `A crowd from ${crowd} networks` : "—"
}

export function sourceOf(source: RuleChangeSource): string {
    switch (source) {
        case RuleChangeSource.INTERNAL:
            return "Seasons"
        case RuleChangeSource.OPERATOR:
            return "An operator"
        case RuleChangeSource.UNSPECIFIED:
            return "Unknown"
    }
}

// A request of its own, so a press sent twice is applied once.
export async function applyRuleSet(
    planet: PlanetAdmin, ruleSet: string, requestId: string = crypto.randomUUID(),
): Promise<void> {
    await planet.applyRuleSet({requestId, ruleSet}, {timeoutMs: LONG_CALL_MS})
}
