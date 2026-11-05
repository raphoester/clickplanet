import {JsonValue} from "@bufbuild/protobuf"

// A card's words with each `{path}` written as the server writes it (the backend's promo placeholders): a number with no
// trailing zero, a duration in its largest whole unit. The page fills a card still being written; a saved card is
// shown as the server filled it.
export type Filled = {text: string, problems: string[]}

const PLACEHOLDER = /\{([^{}]*)\}/g
const DURATION = /^(-?\d+(?:\.\d+)?)s$/

export function fill(words: string, played: JsonValue | undefined, paths: string[]): Filled {
    const problems: string[] = []
    const text = words.replace(PLACEHOLDER, (found, path: string) => {
        if (!paths.includes(path)) {
            problems.push(`{${path}} names no number or duration of the rules.`)
            return found
        }
        const value = valueAt(played, path)
        if (value === undefined) {
            problems.push(`The set plays no value at {${path}}.`)
            return found
        }
        return value
    })
    if (/[{}]/.test(words.replace(PLACEHOLDER, ""))) problems.push("A brace opens or closes no placeholder.")
    return {text, problems}
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
