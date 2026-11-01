import {Duration, JsonObject, JsonValue} from "@bufbuild/protobuf"
import {PlanetAdmin} from "../clients.ts"
import {Change} from "../components/Changes.tsx"
import {secondsWord} from "../components/time.ts"
import {RuleSet} from "../gen/grpc/planet/v1/admin_pb.ts"
import {BoxPace, ClickRules, Rules} from "../gen/grpc/planet/v1/rules_pb.ts"
import {overlay} from "./overlay.ts"

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

// A set as the form edits it: every value as typed. Outside default, "" and undefined are default's value.
export type Draft = {
    name: string, values: Record<string, string>, quiz?: boolean, signInRefill?: boolean, frozen?: boolean,
    allegianceLocked?: boolean, toll?: TollStep[]
}

export function draftOf(set: RuleSet): Draft {
    const rules = set.rules ?? new Rules()
    const json = rules.toJson() as JsonObject

    return {
        name: set.name,
        values: Object.fromEntries(FIELDS.map(field => [field.path, textOf(valueAt(json, field.path))])),
        quiz: rules.bonuses?.quiz?.enabled,
        signInRefill: rules.bonuses?.signInRefill,
        frozen: rules.frozen,
        allegianceLocked: rules.allegianceLocked,
        toll: rules.toll?.steps.map(step => ({
            share: String(Number((step.share * 100).toPrecision(12))), slowdown: String(step.slowdown),
        })),
    }
}

// The rules in force change only through the schedule: a new set starts as a copy of another, with no name. A copy of
// default holds nothing of its own, so it follows default.
export function copyOf(set: RuleSet): Draft {
    return set.name === DEFAULT_SET ? {name: "", values: {}} : {...draftOf(set), name: ""}
}

// Every value the set plays: its own, and default's for the rest. The toll is one value.
export function playedOf(set: RuleSet, defaults: RuleSet | undefined): RuleSet {
    if (defaults === undefined || set.name === DEFAULT_SET) return set
    const base = defaults.rules?.toJson() ?? {}
    return new RuleSet({name: set.name, rules: Rules.fromJson(overlay(base, set.rules?.toJson() ?? {}))})
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

// A name typed, and every value a number; default holds every value. The server holds the rules a set must keep.
export function complete(draft: Draft): boolean {
    const all = draft.name.trim() === DEFAULT_SET
    const typed = FIELDS.map(field => (draft.values[field.path] ?? "").trim())
    const steps = (draft.toll ?? []).flatMap(step => [step.share.trim(), step.slowdown.trim()])
    return draft.name.trim() !== ""
        && typed.every(value => (value === "" && !all) || (value !== "" && Number.isFinite(Number(value))))
        && steps.every(value => value !== "" && Number.isFinite(Number(value)))
}

export const SWITCHES = [
    {key: "frozen", label: "Frozen"},
    {key: "allegianceLocked", label: "Flags locked"},
    {key: "quiz", label: "Quizzes on"},
    {key: "signInRefill", label: "Refill at first sign-in"},
] as const

// Every value the draft holds otherwise than `before`, the set it started from.
export function changesOf(before: Draft, after: Draft): Change[] {
    const changes: Change[] = []
    for (const {key, label} of SWITCHES) {
        if (before[key] !== after[key]) changes.push({label, before: yesOf(before[key]), after: yesOf(after[key])})
    }
    for (const group of GROUPS) {
        for (const field of group.fields) {
            if (sameNumber(before.values[field.path], after.values[field.path])) continue
            changes.push({
                label: `${group.title} · ${field.label}`,
                before: shownOf(field, before.values[field.path]) || "default",
                after: shownOf(field, after.values[field.path]) || "default",
            })
        }
    }
    if (tollOf(before.toll) !== tollOf(after.toll)) changes.push({label: "Toll", before: tollOf(before.toll), after: tollOf(after.toll)})
    return changes
}

export function tollOf(steps: TollStep[] | undefined): string {
    if (steps === undefined) return "default"
    return steps.length === 0 ? "none" : steps.map(step => `${step.share}% ×${step.slowdown}`).join(", ")
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

export function yesOf(on: boolean | undefined): string {
    return on === undefined ? "default" : on ? "yes" : "no"
}

// The values the set holds, and none of default's.
export function ruleSetOf(draft: Draft): RuleSet {
    const json: JsonObject = {}
    const switches = {
        "frozen": draft.frozen, "allegianceLocked": draft.allegianceLocked, "bonuses.quiz.enabled": draft.quiz,
        "bonuses.signInRefill": draft.signInRefill,
    }
    for (const [path, on] of Object.entries(switches)) {
        if (on !== undefined) put(json, path, on)
    }
    for (const field of FIELDS) {
        const typed = (draft.values[field.path] ?? "").trim()
        if (typed !== "") put(json, field.path, jsonOf(Number(typed), field))
    }
    if (draft.toll !== undefined) {
        json.toll = {steps: draft.toll.map(step => ({share: Number(step.share) / 100, slowdown: Number(step.slowdown)}))}
    }

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
