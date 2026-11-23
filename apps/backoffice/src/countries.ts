import {listedFlag} from "./flags.ts"

// The game's country ids are ISO 3166 codes in lower case, a few of its own that no standard names, and r- and ten
// letters or digits for a flag opened since, which the server's list names. The server refuses any other id.
const OWN: Record<string, string> = {
    "gb-eng": "England",
    "gb-sct": "Scotland",
    "gb-wls": "Wales",
    "xb": "Brittany",
}

const NAMES = new Intl.DisplayNames(["en"], {type: "region", fallback: "none"})

const ISO = /^[a-z]{2}$/

// Of the game's own ids, the ones Unicode draws a flag for: England, Scotland and Wales, and not Brittany.
const SUBDIVISIONS = new Set(["gb-eng", "gb-sct", "gb-wls"])

const REGIONAL_A = 0x1f1e6
const BLACK_FLAG = 0x1f3f4
const TAG = 0xe0000
const CANCEL_TAG = 0xe007f

export function countryIdOf(typed: string): string {
    return typed.trim().toLowerCase()
}

export function countryName(typed: string): string | undefined {
    const id = countryIdOf(typed)
    if (id in OWN) return OWN[id]
    if (!ISO.test(id)) return listedFlag(id)?.name
    return NAMES.of(id.toUpperCase()) ?? listedFlag(id)?.name
}

// A country as a sentence names it: its name, or its id when this page has none.
export function countryLabel(id: string): string {
    return countryName(id) ?? id.toUpperCase()
}

// A country's flag as an emoji, or "" for an id Unicode has no flag for. A system with no flag emojis (Windows)
// draws two letters instead, so a flag always goes beside its country's name.
export function flagOf(id: string): string {
    if (SUBDIVISIONS.has(id)) return String.fromCodePoint(BLACK_FLAG, ...codesOf(id.replace("-", ""), TAG), CANCEL_TAG)
    if (id in OWN || !ISO.test(id) || NAMES.of(id.toUpperCase()) === undefined) return ""
    return String.fromCodePoint(...codesOf(id, REGIONAL_A - "a".charCodeAt(0)))
}

// A flag with no emoji but Brittany's, one opened since or one this page does not know, is drawn as a plain grey flag.
export function blankFlag(id: string): boolean {
    return id !== "" && !(id in OWN) && flagOf(id) === ""
}

function codesOf(letters: string, offset: number): number[] {
    return [...letters].map(letter => offset + letter.charCodeAt(0))
}
