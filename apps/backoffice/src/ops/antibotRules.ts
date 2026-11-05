import {FieldInfo, JsonObject, JsonValue, MessageType, ScalarType} from "@bufbuild/protobuf"
import {AntibotAdmin} from "../clients.ts"
import {Change} from "../components/Changes.tsx"
import {secondsWord} from "../components/time.ts"
import {RuleSet} from "../gen/grpc/antibot/v1/admin_pb.ts"
import {Rules} from "../gen/grpc/antibot/v1/rules_pb.ts"
import {overlay} from "./overlay.ts"

// A value of a set, by its path in the set's protojson. A duration is typed in seconds. A value left empty is left
// unset: a set other than default plays default's.
export type Leaf = {
    path: string
    label: string
    kind: "switch" | "number" | "whole" | "seconds" | "secondsList"
}

// A part of the set: a watchdog, the bans, the jury, or a part of one. Its `enabled` is its switch, shown first.
export type Group = {path: string, label: string, leaves: Leaf[], groups: Group[]}

// The form is read off antibot.v1.Rules: a value the proto gains shows here with no edit.
export const PARTS: Group[] = groupsOf(Rules, "")

function groupsOf(type: MessageType, prefix: string): Group[] {
    return type.fields.list()
        .filter(field => field.kind === "message" && !isDuration(field))
        .map(field => {
            const path = `${prefix}${field.jsonName}`
            const T = (field as FieldInfo & {kind: "message"}).T
            return {path, label: labelOf(field.jsonName), leaves: leavesOf(T, `${path}.`), groups: groupsOf(T, `${path}.`)}
        })
}

function leavesOf(type: MessageType, prefix: string): Leaf[] {
    return type.fields.list().flatMap((field): Leaf[] => {
        const path = `${prefix}${field.jsonName}`
        const label = labelOf(field.jsonName)
        switch (field.kind) {
            case "message":
                if (!isDuration(field)) return []
                return [{path, label, kind: field.repeated ? "secondsList" : "seconds"}]
            case "scalar":
                return [{path, label, kind: kindOf(field.T)}]
            case "enum":
            case "map":
                throw new Error(`antibot.v1.Rules gained a ${field.kind} at ${path}: the form has no field for it`)
        }
    }).sort((a, b) => Number(b.path.endsWith(".enabled") || b.path.endsWith(".enforce"))
        - Number(a.path.endsWith(".enabled") || a.path.endsWith(".enforce")))
}

function kindOf(scalar: ScalarType): Leaf["kind"] {
    switch (scalar) {
        case ScalarType.BOOL:
            return "switch"
        case ScalarType.DOUBLE:
        case ScalarType.FLOAT:
            return "number"
        case ScalarType.UINT32:
        case ScalarType.INT32:
        case ScalarType.SINT32:
        case ScalarType.FIXED32:
        case ScalarType.SFIXED32:
            return "whole"
        case ScalarType.INT64:
        case ScalarType.UINT64:
        case ScalarType.FIXED64:
        case ScalarType.SFIXED64:
        case ScalarType.SINT64:
        case ScalarType.STRING:
        case ScalarType.BYTES:
            throw new Error(`antibot.v1.Rules gained a ${ScalarType[scalar]} value: the form has no field for it`)
    }
}

function isDuration(field: FieldInfo): boolean {
    return field.kind === "message" && field.T.typeName === "google.protobuf.Duration"
}

// "minReactions" → "Min reactions".
function labelOf(jsonName: string): string {
    const words = jsonName.replace(/([A-Z])/g, " $1").toLowerCase().trim()
    return words.charAt(0).toUpperCase() + words.slice(1)
}

export const LEAVES: Leaf[] = PARTS.flatMap(function all(group: Group): Leaf[] {
    return [...group.leaves, ...group.groups.flatMap(all)]
})

// Where a leaf sits, as a change names it: "Metronome · Shape · Max gap".
export function placeOf(leaf: Leaf): string {
    return leaf.path.split(".").map(labelOf).join(" · ")
}

export const DEFAULT_SET = "default"

// A switch left unset is undefined, a number or a duration "", a ladder [].
export type Value = string | boolean | string[] | undefined

// A set as the form edits it: every value as typed, by its path.
export type Draft = {name: string, values: Record<string, Value>}

export function draftOf(set: RuleSet): Draft {
    const json = (set.rules ?? new Rules()).toJson() as JsonObject
    return {name: set.name, values: Object.fromEntries(LEAVES.map(leaf => [leaf.path, typedOf(leaf, valueAt(json, leaf.path))]))}
}

// The rules in force change only through the schedule: a new set starts as a copy of another, with no name. A copy of
// default holds nothing of its own, so it follows default.
export function copyOf(set: RuleSet): Draft {
    return {...draftOf(set.name === DEFAULT_SET ? new RuleSet() : set), name: ""}
}

// Every value the set judges with: its own, and default's for the rest.
export function playedOf(set: RuleSet, defaults: RuleSet | undefined): RuleSet {
    if (defaults === undefined || set.name === DEFAULT_SET) return set
    const base = defaults.rules?.toJson() ?? {}
    return new RuleSet({name: set.name, rules: Rules.fromJson(overlay(base, set.rules?.toJson() ?? {}))})
}

function typedOf(leaf: Leaf, value: JsonValue | undefined): Value {
    switch (leaf.kind) {
        case "switch":
            return typeof value === "boolean" ? value : undefined
        case "number":
        case "whole":
            return typeof value === "number" ? String(value) : ""
        case "seconds":
            return typeof value === "string" ? value.replace(/s$/, "") : ""
        case "secondsList":
            return Array.isArray(value) ? value.map(each => String(each).replace(/s$/, "")) : []
    }
}

function valueAt(json: JsonObject, path: string): JsonValue | undefined {
    let value: JsonValue | undefined = json
    for (const key of path.split(".")) {
        value = typeof value === "object" && value !== null && !Array.isArray(value) ? value[key] : undefined
    }
    return value
}

// Why the draft is no set, or undefined when it is one. Default keeps every value it held when the form opened (`kept`);
// the server holds the bounds a set must keep.
export function flawOf(draft: Draft, kept: Draft): string | undefined {
    const all = draft.name.trim() === DEFAULT_SET
    for (const leaf of LEAVES) {
        const value = draft.values[leaf.path]
        const typed = Array.isArray(value) ? value : typeof value === "string" ? [value] : []
        if (leaf.kind === "secondsList" && typed.some(each => each.trim() === "")) return `${placeOf(leaf)}: a step is empty.`
        if (all && !unset(kept.values[leaf.path]) && unset(value)) return `${placeOf(leaf)}: default needs a value.`
        for (const each of typed) {
            if (each.trim() !== "" && !Number.isFinite(Number(each))) return `${placeOf(leaf)}: ${each} is not a number.`
        }
    }
    return undefined
}

export function unset(value: Value): boolean {
    return value === undefined || (typeof value === "string" && value.trim() === "") || (Array.isArray(value) && value.length === 0)
}

export function rulesOf(draft: Draft): Rules {
    const json: JsonObject = {}
    for (const leaf of LEAVES) {
        const written = jsonOf(leaf, draft.values[leaf.path])
        if (written !== undefined) put(json, leaf.path, written)
    }
    return Rules.fromJson(json)
}

function jsonOf(leaf: Leaf, value: Value): JsonValue | undefined {
    switch (leaf.kind) {
        case "switch":
            return typeof value === "boolean" ? value : undefined
        case "number":
        case "whole": {
            const typed = typeof value === "string" ? value.trim() : ""
            if (typed === "") return undefined
            return leaf.kind === "whole" ? Math.max(0, Math.round(Number(typed))) : Number(typed)
        }
        case "seconds": {
            const typed = typeof value === "string" ? value.trim() : ""
            return typed === "" ? undefined : `${Number(typed)}s`
        }
        case "secondsList":
            return Array.isArray(value) && value.length > 0 ? value.map(each => `${Number(each)}s`) : undefined
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

// A value as the operator reads it: a duration in the unit it reads best in. A value unset reads "".
export function shownOf(leaf: Leaf, value: Value): string {
    switch (leaf.kind) {
        case "switch":
            return value === undefined ? "" : value ? "on" : "off"
        case "number":
        case "whole":
            return typeof value === "string" ? numberOf(value) : ""
        case "seconds":
            return typeof value === "string" ? secondsOf(value) : ""
        case "secondsList":
            return (Array.isArray(value) ? value : []).map(secondsOf).join(", ")
    }
}

// "0.60" and "0.6" are one value.
function numberOf(typed: string): string {
    const text = typed.trim()
    return text === "" || !Number.isFinite(Number(text)) ? text : String(Number(text))
}

function secondsOf(typed: string): string {
    const text = typed.trim()
    return text === "" || !Number.isFinite(Number(text)) ? text : secondsWord(Number(text))
}

// Every value the draft holds otherwise than `before`, the set it started from.
export function changesOf(before: Draft, after: Draft): Change[] {
    const unsetWord = after.name.trim() === DEFAULT_SET ? "unset" : "default"
    const shown = (leaf: Leaf, value: Value) => shownOf(leaf, value) || unsetWord
    return LEAVES
        .filter(leaf => changedAt(before, after, leaf))
        .map(leaf => ({label: placeOf(leaf), before: shown(leaf, before.values[leaf.path]), after: shown(leaf, after.values[leaf.path])}))
}

export function changedAt(before: Draft, after: Draft, leaf: Leaf): boolean {
    return shownOf(leaf, before.values[leaf.path]) !== shownOf(leaf, after.values[leaf.path])
}

export async function saveRuleSet(antibot: AntibotAdmin, draft: Draft): Promise<void> {
    await antibot.saveRuleSet({set: new RuleSet({name: draft.name.trim(), rules: rulesOf(draft)})})
}

export function bansOf(set: RuleSet): string {
    return set.rules?.bans?.enforce === true ? "Enforced" : "Not enforced"
}

// A watchdog's part of a set is the one with an enabled switch.
const WATCHDOGS = Rules.fields.list()
    .filter(field => field.kind === "message" && field.T.fields.findJsonName("enabled") !== undefined)
    .map(field => field.localName)

export function offOf(set: RuleSet): string {
    const parts = (set.rules ?? new Rules()) as unknown as Record<string, {enabled: boolean} | undefined>
    const off = WATCHDOGS.filter(watchdog => parts[watchdog]?.enabled !== true)
    return off.length === 0 ? "None" : off.join(", ")
}

export function crowdOf(set: RuleSet): string {
    return `${set.rules?.cohort?.certainMembers ?? 0} scopes`
}
