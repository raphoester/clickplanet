import {Duration, JsonObject, JsonValue} from "@bufbuild/protobuf"
import {PlanetAdmin} from "../clients.ts"
import {Change} from "../components/Changes.tsx"
import {secondsWord} from "../components/time.ts"
import {RuleSet} from "../gen/grpc/planet/v1/admin_pb.ts"
import {BoxPace, ClickRules, Rules} from "../gen/grpc/planet/v1/rules_pb.ts"

export const DEFAULT_SET = "default"

export function tempoOf(set: RuleSet): string {
    return `${set.rules?.clicks?.tempo ?? 0}×`
}

// A frozen set takes no play: no click, bomb, shield, refill, box or quiz.
export function playOf(set: RuleSet): string {
    return set.rules?.frozen ? "Frozen" : "Open"
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
        {path: "clicks.tempo", label: "Refill, times as fast", kind: "number"},
        {path: "clicks.perSecond", label: "Clicks a second", kind: "number"},
        {path: "clicks.burst", label: "Clicks in hand", kind: "whole"},
        {path: "clicks.newAccountClicks", label: "New account, clicks in hand", kind: "number"},
        {path: "clicks.linkedMultiplier", label: "Signed in, times as fast", kind: "number"},
        {path: "clicks.scopeMultiplier", label: "Address, in accounts", kind: "number"},
        {path: "clicks.v6ScopeMultiplier", label: "IPv6 address, in accounts", kind: "number"},
        {path: "clicks.guestScopeMultiplier", label: "Guests of an address, in accounts", kind: "number"},
    ]},
    {title: "Boxes", fields: [
        {path: "bonuses.boxes.minInterval", label: "Shortest box wait (s)", kind: "seconds"},
        {path: "bonuses.boxes.maxInterval", label: "Longest box wait (s)", kind: "seconds"},
        {path: "bonuses.boxes.maxChargesPerHour", label: "Charges an hour, at most", kind: "whole"},
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
    {title: "Rewarded ad", fields: [
        {path: "bonuses.adRefill.every", label: "Once every (s)", kind: "seconds"},
    ]},
    {title: "Quizzes", fields: [
        {path: "bonuses.quiz.minInterval", label: "Shortest quiz wait (s)", kind: "seconds"},
        {path: "bonuses.quiz.maxInterval", label: "Longest quiz wait (s)", kind: "seconds"},
        {path: "bonuses.quiz.maxChargesPerHour", label: "Quizzes won an hour, at most", kind: "whole"},
        {path: "bonuses.quiz.offerTtl", label: "Banner (s)", kind: "seconds"},
        {path: "bonuses.quiz.answerWindow", label: "Time to answer (s)", kind: "seconds"},
        {path: "bonuses.quiz.missRetry", label: "Next quiz after a miss (s)", kind: "seconds"},
        {path: "bonuses.quiz.leaderBias", label: "Lean to the leaders", kind: "number"},
    ]},
]

const FIELDS = GROUPS.flatMap(group => group.fields)

// A toll step as the form edits it: the share in percent.
export type TollStep = {share: string, slowdown: string}

// A set as the form edits it: every value as typed.
export type Draft = {
    name: string, values: Record<string, string>, quiz: boolean, signInRefill: boolean, adRefill: boolean, frozen: boolean,
    toll: TollStep[],
}

export function draftOf(set: RuleSet): Draft {
    const rules = set.rules ?? new Rules()
    const json = rules.toJson({emitDefaultValues: true}) as JsonObject

    return {
        name: set.name,
        values: Object.fromEntries(FIELDS.map(field => [field.path, textOf(valueAt(json, field.path))])),
        quiz: rules.bonuses?.quiz?.enabled ?? false,
        signInRefill: rules.bonuses?.signInRefill ?? false,
        adRefill: rules.bonuses?.adRefill?.enabled ?? false,
        frozen: rules.frozen,
        toll: (rules.toll?.steps ?? []).map(step => ({
            share: String(Number((step.share * 100).toPrecision(12))), slowdown: String(step.slowdown),
        })),
    }
}

// The rules in force change only through the schedule: a new set starts as a copy of another, with no name.
export function copyOf(set: RuleSet): Draft {
    return {...draftOf(set), name: ""}
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

// Every value the draft holds otherwise than `before`, the set it started from.
export function changesOf(before: Draft, after: Draft): Change[] {
    const changes: Change[] = []
    if (before.frozen !== after.frozen) changes.push({label: "Frozen", before: yesOf(before.frozen), after: yesOf(after.frozen)})
    if (before.quiz !== after.quiz) changes.push({label: "Quizzes on", before: yesOf(before.quiz), after: yesOf(after.quiz)})
    if (before.signInRefill !== after.signInRefill) {
        changes.push({label: "Refill at first sign-in", before: yesOf(before.signInRefill), after: yesOf(after.signInRefill)})
    }
    if (before.adRefill !== after.adRefill) {
        changes.push({label: "Refill for a rewarded ad", before: yesOf(before.adRefill), after: yesOf(after.adRefill)})
    }
    for (const group of GROUPS) {
        for (const field of group.fields) {
            if (sameNumber(before.values[field.path], after.values[field.path])) continue
            changes.push({
                label: `${group.title} · ${field.label}`,
                before: shownOf(field, before.values[field.path]),
                after: shownOf(field, after.values[field.path]),
            })
        }
    }
    const tollOf = (draft: Draft) => draft.toll.map(step => `${step.share}% ×${step.slowdown}`).join(", ")
    if (tollOf(before) !== tollOf(after)) changes.push({label: "Toll", before: tollOf(before), after: tollOf(after)})
    return changes
}

// Whether a field holds otherwise than `before`: "3" and "3.0" are one value.
export function changedAt(before: Draft, after: Draft, path: string): boolean {
    return !sameNumber(before.values[path], after.values[path])
}

// A value as the operator reads it: a duration in the unit it reads best in.
export function shownOf(field: Field, typed: string): string {
    if (field.kind !== "seconds" || typed.trim() === "" || !Number.isFinite(Number(typed))) return typed
    return secondsWord(Number(typed))
}

function sameNumber(a: string | undefined, b: string | undefined): boolean {
    const x = (a ?? "").trim()
    const y = (b ?? "").trim()
    return x === y || (x !== "" && y !== "" && Number(x) === Number(y))
}

function yesOf(on: boolean): string {
    return on ? "yes" : "no"
}

export function ruleSetOf(draft: Draft): RuleSet {
    const json: JsonObject = {
        bonuses: {quiz: {enabled: draft.quiz}, signInRefill: draft.signInRefill, adRefill: {enabled: draft.adRefill}},
        frozen: draft.frozen,
    }
    for (const field of FIELDS) {
        put(json, field.path, jsonOf(Number(draft.values[field.path]), field))
    }
    json.toll = {steps: draft.toll.map(step => ({share: Number(step.share) / 100, slowdown: Number(step.slowdown)}))}

    return new RuleSet({name: draft.name.trim(), rules: Rules.fromJson(json)})
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
