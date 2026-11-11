import {JsonObject, JsonValue} from "@bufbuild/protobuf"
import {secondsWord} from "../components/time.ts"

// A value of a set's protojson, by its path, as the game rules and the box and quiz rules type it: a duration in
// seconds, a count whole.
export type Field = {path: string, label: string, kind: "number" | "whole" | "seconds"}

// A switch of a set, by its path: a box in default, a choice of default's, yes or no elsewhere. `short` names it in
// what a save changes.
export type Switch = {path: string, label: string, short: string}

export function valueAt(json: JsonObject, path: string): JsonValue | undefined {
    let value: JsonValue | undefined = json
    for (const key of path.split(".")) {
        value = typeof value === "object" && value !== null && !Array.isArray(value) ? value[key] : undefined
    }
    return value
}

// A value as the form types it: a duration's seconds, a number's digits, "" for a value the set leaves unset.
export function typedOf(value: JsonValue | undefined): string {
    if (typeof value === "string") return value.replace(/s$/, "")
    if (typeof value === "number") return String(value)
    return ""
}

export function jsonOf(value: number, field: Field): JsonValue {
    switch (field.kind) {
        case "seconds":
            return `${value}s`
        case "whole":
            return Math.max(0, Math.round(value))
        case "number":
            return value
    }
}

export function put(json: JsonObject, path: string, value: JsonValue) {
    const keys = path.split(".")
    let at = json
    for (const key of keys.slice(0, -1)) {
        const next = at[key]
        at = typeof next === "object" && next !== null && !Array.isArray(next) ? next : (at[key] = {})
    }
    at[keys[keys.length - 1]] = value
}

// "3" and "3.0" are one value.
export function sameNumber(a: string | undefined, b: string | undefined): boolean {
    const x = (a ?? "").trim()
    const y = (b ?? "").trim()
    return x === y || (x !== "" && y !== "" && Number(x) === Number(y))
}

// Whether a field holds otherwise than `before`.
export function changedAt(before: {values: Record<string, string>}, after: {values: Record<string, string>}, path: string): boolean {
    return !sameNumber(before.values[path], after.values[path])
}

// Every value typed is a number; in default (`all`), none is left empty.
export function numbersTyped(fields: Field[], values: Record<string, string>, all: boolean): boolean {
    return fields
        .map(field => (values[field.path] ?? "").trim())
        .every(value => (value === "" && !all) || (value !== "" && Number.isFinite(Number(value))))
}

// A value as the operator reads it: a duration in the unit it reads best in.
export function shownOf(field: Field, typed: string): string {
    if (field.kind !== "seconds" || typed.trim() === "" || !Number.isFinite(Number(typed))) return typed
    return secondsWord(Number(typed))
}

export function yesOf(on: boolean | undefined): string {
    return on === undefined ? "default" : on ? "yes" : "no"
}
