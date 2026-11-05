import {JsonValue} from "@bufbuild/protobuf"
import {Placeholder} from "../gen/grpc/promo/v1/admin_pb.ts"

// A card's words with each `{path}` written as the server writes it (the backend's promo placeholders): a number with no
// trailing zero, a duration in its largest whole unit. The page fills a card still being written; a saved card is
// shown as the server filled it.
export type Filled = {text: string, problems: string[]}

// What a card's words may name: the values the set plays, by their path in the rules, and the example battle's, as the
// server wrote them.
export type Values = {played: JsonValue | undefined, rules: string[], battle: Placeholder[]}

const PLACEHOLDER = /\{([^{}]*)\}/g
const DURATION = /^(-?\d+(?:\.\d+)?)s$/

export function fill(words: string, values: Values): Filled {
    const problems: string[] = []
    const text = words.replace(PLACEHOLDER, (found, path: string) => {
        const battle = values.battle.find(placeholder => placeholder.path === path)
        if (battle !== undefined) {
            if (battle.example !== "") return battle.example
            problems.push(`The battle has no value at {${path}}.`)
            return found
        }
        if (!values.rules.includes(path)) {
            problems.push(`{${path}} names no number or duration of the rules or the battle.`)
            return found
        }
        const value = valueAt(values.played, path)
        if (value === undefined) {
            problems.push(`The set plays no value at {${path}}.`)
            return found
        }
        return value
    })
    if (/[{}]/.test(words.replace(PLACEHOLDER, ""))) problems.push("A brace opens or closes no placeholder.")
    return {text, problems}
}

export function namesBattle(words: string, battle: Placeholder[]): boolean {
    return [...words.matchAll(PLACEHOLDER)].some(([, path]) => battle.some(placeholder => placeholder.path === path))
}

function valueAt(played: JsonValue | undefined, path: string): string | undefined {
    let value = played
    for (const segment of path.split(".")) {
        if (typeof value !== "object" || value === null || Array.isArray(value)) return undefined
        value = value[segment]
    }
    if (typeof value === "number") return String(value)
    if (typeof value === "string") {
        const seconds = DURATION.exec(value)
        if (seconds !== null) return durationWord(Math.round(Number(seconds[1]) * 1000))
    }
    return undefined
}

// "1 h", "2 min", "90 s", "500 ms": the largest unit that counts it whole.
export function durationWord(ms: number): string {
    if (ms === 0) return "0 s"
    for (const [size, name] of [[3_600_000, "h"], [60_000, "min"], [1000, "s"]] as const) {
        if (ms % size === 0) return `${ms / size} ${name}`
    }
    return `${ms} ms`
}
