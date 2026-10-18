// The game's country ids are ISO 3166 codes in lower case, and a few of its own that no standard names. The
// server holds the list and refuses any other id; this only names what the operator typed.
const OWN: Record<string, string> = {
    "gb-eng": "England",
    "gb-sct": "Scotland",
    "gb-wls": "Wales",
    "xb": "Brittany",
}

const NAMES = new Intl.DisplayNames(["en"], {type: "region", fallback: "none"})

export function countryIdOf(typed: string): string {
    return typed.trim().toLowerCase()
}

export function countryName(typed: string): string | undefined {
    const id = countryIdOf(typed)
    if (id in OWN) return OWN[id]
    if (!/^[a-z]{2}$/.test(id)) return undefined
    return NAMES.of(id.toUpperCase())
}

// A country as a sentence names it: its name, or its id when this page has none.
export function countryLabel(id: string): string {
    return countryName(id) ?? id.toUpperCase()
}
