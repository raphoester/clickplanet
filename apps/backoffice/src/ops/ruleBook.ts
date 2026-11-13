import {JsonObject, JsonValue} from "@bufbuild/protobuf"
import {Change} from "../components/Changes.tsx"
import {overlay} from "./overlay.ts"
import {DEFAULT_SET} from "./rules.ts"
import {Field, jsonOf, numbersTyped, put, sameNumber, shownOf, Switch, typedOf, valueAt, yesOf} from "./ruleValues.ts"

// A module's rule sets as one form edits them, by the paths of their protojson: the offers' (bonusRules.ts) and the
// quizzes' (quizRules.ts).

// A rule set as a module keeps it: its name, and its rules.
export type Kept = {name: string, rules?: {toJson: () => JsonValue}}

// A group of the form: its switches first, then a field per value.
export type Group = {title: string, switches: Switch[], fields: Field[]}

// What the form knows of a module: the switch that freezes a set, the groups of its values, and a set made from a name
// and its protojson.
export type Book<S extends Kept> = {frozen: Switch, groups: Group[], setOf: (name: string, json: JsonValue) => S}

// A set as the form edits it: every value as typed, every switch, by its path. Outside default, "" and undefined are
// default's value.
export type Draft = {name: string, values: Record<string, string>, switches: Record<string, boolean | undefined>}

function fieldsOf<S extends Kept>(book: Book<S>): Field[] {
    return book.groups.flatMap(group => group.fields)
}

function switchesOf<S extends Kept>(book: Book<S>): Switch[] {
    return [book.frozen, ...book.groups.flatMap(group => group.switches)]
}

export function draftOf<S extends Kept>(book: Book<S>, set: Kept): Draft {
    const json = (set.rules?.toJson() ?? {}) as JsonObject
    return {
        name: set.name,
        values: Object.fromEntries(fieldsOf(book).map(field => [field.path, typedOf(valueAt(json, field.path))])),
        switches: Object.fromEntries(switchesOf(book).map(({path}) => {
            const on = valueAt(json, path)
            return [path, typeof on === "boolean" ? on : undefined]
        })),
    }
}

// A module plays the schedule's set: a new one starts as a copy of another, with no name. A copy of default holds
// nothing of its own, so it follows default.
export function copyOf<S extends Kept>(book: Book<S>, set: S): Draft {
    return {...draftOf(book, set.name === DEFAULT_SET ? {name: ""} : set), name: ""}
}

// Every value the set plays: its own, and default's for the rest.
export function playedOf<S extends Kept>(book: Book<S>, set: S, defaults: S | undefined): S {
    if (defaults === undefined || set.name === DEFAULT_SET) return set
    return book.setOf(set.name, overlay(defaults.rules?.toJson() ?? {}, set.rules?.toJson() ?? {}))
}

// A name typed, and every value a number; default holds every value. The server holds the rules a set must keep.
export function complete<S extends Kept>(book: Book<S>, draft: Draft): boolean {
    return draft.name.trim() !== "" && numbersTyped(fieldsOf(book), draft.values, draft.name.trim() === DEFAULT_SET)
}

// Every value the draft holds otherwise than `before`, the set it started from, in the form's order.
export function changesOf<S extends Kept>(book: Book<S>, before: Draft, after: Draft): Change[] {
    const changes: Change[] = []
    const switched = (each: Switch) => {
        const was = before.switches[each.path]
        const is = after.switches[each.path]
        if (was !== is) changes.push({label: each.short, before: yesOf(was), after: yesOf(is)})
    }
    switched(book.frozen)
    for (const group of book.groups) {
        group.switches.forEach(switched)
        for (const field of group.fields) {
            if (sameNumber(before.values[field.path], after.values[field.path])) continue
            changes.push({
                label: `${group.title} · ${field.label}`,
                before: shownOf(field, before.values[field.path]) || "default",
                after: shownOf(field, after.values[field.path]) || "default",
            })
        }
    }
    return changes
}

// The values the set holds, and none of default's.
export function ruleSetOf<S extends Kept>(book: Book<S>, draft: Draft): S {
    const json: JsonObject = {}
    for (const {path} of switchesOf(book)) {
        const on = draft.switches[path]
        if (on !== undefined) put(json, path, on)
    }
    for (const field of fieldsOf(book)) {
        const typed = (draft.values[field.path] ?? "").trim()
        if (typed !== "") put(json, field.path, jsonOf(Number(typed), field))
    }
    return book.setOf(draft.name.trim(), json)
}
