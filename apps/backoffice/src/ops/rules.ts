import {Duration, JsonObject, JsonValue} from "@bufbuild/protobuf"
import {PlanetAdmin} from "../clients.ts"
import {BoxPace, ClickRules, RuleSet} from "../gen/grpc/planet/v1/admin_pb.ts"

export const DEFAULT_SET = "default"

export function tempoOf(set: RuleSet): string {
    return `${set.tempo}×`
}

export function clicksOf(clicks: ClickRules | undefined): string {
    if (clicks === undefined) return "—"

    return `${clicks.burst} in hand, ${clicks.perSecond} a second`
}

export function boxesOf(boxes: BoxPace | undefined): string {
    if (boxes === undefined) return "—"

    const shortest = spanOf(boxes.minInterval)
    const longest = spanOf(boxes.maxInterval)
    const every = shortest === longest ? shortest : `${shortest} to ${longest}`
    return `Every ${every}, ${boxes.maxChargesPerHour} an hour at most`
}

function spanOf(duration: Duration | undefined): string {
    const seconds = Number(duration?.seconds ?? 0n)
    return seconds % 60 === 0 ? `${seconds / 60} min` : `${seconds} s`
}

// A value of the set's JSON, by its path: a duration is typed in seconds, a count is whole.
export type Field = {path: string, label: string, kind: "number" | "whole" | "seconds"}

export const GROUPS: {title: string, fields: Field[]}[] = [
    {title: "Clicks", fields: [
        {path: "tempo", label: "Refill, times as fast", kind: "number"},
        {path: "clicks.perSecond", label: "Clicks a second", kind: "number"},
        {path: "clicks.burst", label: "Clicks in hand", kind: "whole"},
        {path: "clicks.newAccountClicks", label: "New account, clicks in hand", kind: "number"},
        {path: "clicks.linkedMultiplier", label: "Signed in, times as fast", kind: "number"},
        {path: "clicks.scopeMultiplier", label: "Address, in accounts", kind: "number"},
        {path: "clicks.v6ScopeMultiplier", label: "IPv6 address, in accounts", kind: "number"},
        {path: "clicks.guestScopeMultiplier", label: "Guests of an address, in accounts", kind: "number"},
    ]},
    {title: "Boxes", fields: [
        {path: "boxes.minInterval", label: "Shortest box wait (s)", kind: "seconds"},
        {path: "boxes.maxInterval", label: "Longest box wait (s)", kind: "seconds"},
        {path: "boxes.maxChargesPerHour", label: "Charges an hour, at most", kind: "whole"},
        {path: "bonuses.offerTtl", label: "Box flight (s)", kind: "seconds"},
        {path: "bonuses.missRetry", label: "Next box after a miss (s)", kind: "seconds"},
        {path: "bonuses.activeWithin", label: "Last click within (s)", kind: "seconds"},
    ]},
    {title: "What a box is", fields: [
        {path: "bonuses.kinds.refill", label: "Refill, weight", kind: "number"},
        {path: "bonuses.kinds.spreadClicks", label: "Spread, weight", kind: "number"},
        {path: "bonuses.kinds.encloseClicks", label: "Enclose, weight", kind: "number"},
        {path: "bonuses.kinds.bomb", label: "Bomb, weight", kind: "number"},
        {path: "bonuses.kinds.shields", label: "Shields, weight", kind: "number"},
    ]},
    {title: "Charges", fields: [
        {path: "bonuses.spread.clicks", label: "Spread clicks held", kind: "whole"},
        {path: "bonuses.spread.maxPerBox", label: "Spread clicks a box, at most", kind: "whole"},
        {path: "bonuses.bomb.rings", label: "Bomb rings", kind: "number"},
        {path: "bonuses.enclose.maxTiles", label: "Enclose tiles, at most", kind: "whole"},
        {path: "bonuses.enclose.held", label: "Enclosures held", kind: "whole"},
        {path: "bonuses.enclose.maxPerBox", label: "Enclosures a box, at most", kind: "whole"},
        {path: "bonuses.shield.held", label: "Shields held", kind: "whole"},
        {path: "bonuses.shield.minPerBox", label: "Shields a box, at least", kind: "whole"},
        {path: "bonuses.shield.maxPerBox", label: "Shields a box, at most", kind: "whole"},
        {path: "bonuses.shield.perTile", label: "Shields on a tile, at most", kind: "whole"},
    ]},
    {title: "New accounts", fields: [
        {path: "bonuses.welcome.within", label: "New for (s)", kind: "seconds"},
        {path: "bonuses.welcome.boxAfter", label: "First box after (s)", kind: "seconds"},
    ]},
    {title: "Quizzes", fields: [
        {path: "quiz.minInterval", label: "Shortest quiz wait (s)", kind: "seconds"},
        {path: "quiz.maxInterval", label: "Longest quiz wait (s)", kind: "seconds"},
        {path: "quiz.maxChargesPerHour", label: "Quizzes won an hour, at most", kind: "whole"},
        {path: "quiz.offerTtl", label: "Banner (s)", kind: "seconds"},
        {path: "quiz.answerWindow", label: "Time to answer (s)", kind: "seconds"},
        {path: "quiz.missRetry", label: "Next quiz after a miss (s)", kind: "seconds"},
        {path: "quiz.leaderBias", label: "Lean to the leaders", kind: "number"},
    ]},
]

const FIELDS = GROUPS.flatMap(group => group.fields)

// A toll step as the form edits it: the share in percent.
export type TollStep = {share: string, slowdown: string}

// A set as the form edits it: every value as typed.
export type Draft = {name: string, values: Record<string, string>, quiz: boolean, toll: TollStep[]}

export function draftOf(set: RuleSet, name: string = set.name): Draft {
    const json = set.toJson({emitDefaultValues: true}) as JsonObject

    return {
        name,
        values: Object.fromEntries(FIELDS.map(field => [field.path, textOf(valueAt(json, field.path))])),
        quiz: set.quiz?.enabled ?? false,
        toll: set.tollSteps.map(step => ({share: String(Number((step.share * 100).toPrecision(12))), slowdown: String(step.slowdown)})),
    }
}

function valueAt(json: JsonObject, path: string): JsonValue | undefined {
    let value: JsonValue | undefined = json
    for (const key of path.split(".")) {
        value = typeof value === "object" && value !== null && !Array.isArray(value) ? value[key] : undefined
    }
    return value
}

function textOf(value: JsonValue | undefined): string {
    if (typeof value === "string") return value.replace(/s$/, "")
    if (typeof value === "number") return String(value)
    return ""
}

// Every value a number and a name typed; the server holds the rules a set must keep.
export function complete(draft: Draft): boolean {
    const numbers = [...Object.values(draft.values), ...draft.toll.flatMap(step => [step.share, step.slowdown])]
    return draft.name.trim() !== "" && numbers.every(value => value.trim() !== "" && Number.isFinite(Number(value)))
}

export function ruleSetOf(draft: Draft): RuleSet {
    const json: JsonObject = {name: draft.name.trim(), quiz: {enabled: draft.quiz}}
    for (const field of FIELDS) {
        put(json, field.path, jsonOf(Number(draft.values[field.path]), field))
    }
    json.tollSteps = draft.toll.map(step => ({share: Number(step.share) / 100, slowdown: Number(step.slowdown)}))

    return RuleSet.fromJson(json)
}

function jsonOf(value: number, field: Field): JsonValue {
    switch (field.kind) {
        case "seconds":
            return `${value}s`
        case "whole":
            return Math.max(0, Math.round(value))
        case "number":
            return value
    }
}

function put(json: JsonObject, path: string, value: JsonValue) {
    const keys = path.split(".")
    let at = json
    for (const key of keys.slice(0, -1)) {
        const next = at[key]
        at = typeof next === "object" && next !== null && !Array.isArray(next) ? next : (at[key] = {})
    }
    at[keys[keys.length - 1]] = value
}

export async function saveRuleSet(planet: PlanetAdmin, draft: Draft): Promise<void> {
    await planet.saveRuleSet({set: ruleSetOf(draft)})
}
