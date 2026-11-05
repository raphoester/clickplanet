import {JsonObject, JsonValue} from "@bufbuild/protobuf"

// A set's rules as JSON over default's, as the backend plays them: an object is overlaid key by key, but a message of
// lists only (the toll) is one value, and a list replaces default's.
export function overlay(base: JsonValue, top: JsonValue): JsonValue {
    if (!isObject(base) || !isObject(top) || isLists(base)) return top
    const merged: JsonObject = {...base}
    for (const [key, value] of Object.entries(top)) merged[key] = key in base ? overlay(base[key], value) : value
    return merged
}

function isObject(json: JsonValue): json is JsonObject {
    return typeof json === "object" && json !== null && !Array.isArray(json)
}

function isLists(json: JsonObject): boolean {
    const values = Object.values(json)
    return values.length > 0 && values.every(Array.isArray)
}
