import {Duration, Timestamp} from "@bufbuild/protobuf"
import {PlanetAdmin} from "../clients.ts"
import {BoxPace, GetRulesResponse, RuleSet} from "../gen/grpc/planet/v1/admin_pb.ts"

export const DEFAULT_SET = "default"

export function tempoOf(set: RuleSet): string {
    return set.tempo === 1 ? "As configured" : `${set.tempo}× as fast`
}

export function boxesOf(boxes: BoxPace | undefined): string {
    if (boxes === undefined) return "—"

    const shortest = spanOf(boxes.minInterval)
    const longest = spanOf(boxes.maxInterval)
    const every = shortest === longest ? shortest : `${shortest} to ${longest}`
    return `Every ${every}, ${boxes.maxChargesPerHour} an hour at most`
}

function spanOf(duration: Duration | undefined): string {
    const seconds = secondsOf(duration)
    return seconds % 60 === 0 ? `${seconds / 60} min` : `${seconds} s`
}

function secondsOf(duration: Duration | undefined): number {
    return Number(duration?.seconds ?? 0n)
}

// A set as the form edits it: the waits in seconds, every value as typed.
export type Draft = {name: string, tempo: string, minSeconds: string, maxSeconds: string, cap: string}

export const NEW_DRAFT: Draft = {name: "", tempo: "1", minSeconds: "240", maxSeconds: "480", cap: "12"}

export function draftOf(set: RuleSet): Draft {
    return {
        name: set.name,
        tempo: String(set.tempo),
        minSeconds: String(secondsOf(set.boxes?.minInterval)),
        maxSeconds: String(secondsOf(set.boxes?.maxInterval)),
        cap: String(set.boxes?.maxChargesPerHour ?? 0),
    }
}

// Every value a number and a name typed; the server holds the rules a set must keep.
export function complete(draft: Draft): boolean {
    const numbers = [draft.tempo, draft.minSeconds, draft.maxSeconds, draft.cap]
    return draft.name.trim() !== "" && numbers.every(value => value.trim() !== "" && Number.isFinite(Number(value)))
}

export function ruleSetOf(draft: Draft): RuleSet {
    return new RuleSet({
        name: draft.name.trim(),
        tempo: Number(draft.tempo),
        boxes: new BoxPace({
            minInterval: new Duration({seconds: BigInt(Math.round(Number(draft.minSeconds)))}),
            maxInterval: new Duration({seconds: BigInt(Math.round(Number(draft.maxSeconds)))}),
            maxChargesPerHour: Math.round(Number(draft.cap)),
        }),
    })
}

export async function saveRuleSet(planet: PlanetAdmin, draft: Draft): Promise<void> {
    await planet.saveRuleSet({set: ruleSetOf(draft)})
}

// What the game plays now, as planet decides it: the override while it runs, then the newest baseline, then default.
export type Playing =
    | {layer: "override", set: string, until: Date}
    | {layer: "baseline", set: string, request: string, since: Date | undefined}
    | {layer: "default", set: string}

export function playingAt(rules: GetRulesResponse, now: Date): Playing {
    const override = rules.override
    const starts = override?.startsAt?.toDate()
    const ends = override?.endsAt?.toDate()
    if (override !== undefined && starts !== undefined && ends !== undefined && starts <= now && now < ends) {
        return {layer: "override", set: override.ruleSet, until: ends}
    }

    const [newest] = rules.baselines
    if (newest !== undefined) {
        return {layer: "baseline", set: newest.ruleSet, request: newest.requestId, since: newest.appliedAt?.toDate()}
    }
    return {layer: "default", set: DEFAULT_SET}
}

// The server holds an override to a day at most.
export const MAX_OVERRIDE_MINUTES = 24 * 60

export function overrideMinutesValid(minutes: string): boolean {
    const value = Number(minutes)
    return Number.isInteger(value) && value > 0 && value <= MAX_OVERRIDE_MINUTES
}

export async function overrideRules(planet: PlanetAdmin, set: string, minutes: number, now: Date): Promise<void> {
    await planet.overrideRules({ruleSet: set, until: Timestamp.fromDate(new Date(now.getTime() + minutes * 60_000))})
}

export async function endOverride(planet: PlanetAdmin): Promise<void> {
    await planet.endOverride({})
}
