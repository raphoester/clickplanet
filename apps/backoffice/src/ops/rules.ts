import {Duration} from "@bufbuild/protobuf"
import {PlanetAdmin} from "../clients.ts"
import {AntiBotRules, BoxPace, RuleSet} from "../gen/grpc/planet/v1/admin_pb.ts"

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

export function crowdOf(set: RuleSet): string {
    const crowd = set.antiBot?.crowd ?? 0
    return crowd > 0 ? `A crowd from ${crowd} networks` : "—"
}

function spanOf(duration: Duration | undefined): string {
    const seconds = secondsOf(duration)
    return seconds % 60 === 0 ? `${seconds / 60} min` : `${seconds} s`
}

function secondsOf(duration: Duration | undefined): number {
    return Number(duration?.seconds ?? 0n)
}

// A set as the form edits it: the waits in seconds, every value as typed.
export type Draft = {name: string, tempo: string, minSeconds: string, maxSeconds: string, cap: string, crowd: string}

export const NEW_DRAFT: Draft = {name: "", tempo: "1", minSeconds: "240", maxSeconds: "480", cap: "12", crowd: "6"}

export function draftOf(set: RuleSet): Draft {
    return {
        name: set.name,
        tempo: String(set.tempo),
        minSeconds: String(secondsOf(set.boxes?.minInterval)),
        maxSeconds: String(secondsOf(set.boxes?.maxInterval)),
        cap: String(set.boxes?.maxChargesPerHour ?? 0),
        crowd: String(set.antiBot?.crowd ?? 0),
    }
}

// Every value a number and a name typed; the server holds the rules a set must keep.
export function complete(draft: Draft): boolean {
    const numbers = [draft.tempo, draft.minSeconds, draft.maxSeconds, draft.cap, draft.crowd]
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
        antiBot: new AntiBotRules({crowd: Math.round(Number(draft.crowd))}),
    })
}

export async function saveRuleSet(planet: PlanetAdmin, draft: Draft): Promise<void> {
    await planet.saveRuleSet({set: ruleSetOf(draft)})
}
