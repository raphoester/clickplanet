import {translator} from "../../i18n/page.ts"

export type Value = {
    code: string,
    name: string
}

// Letters that carry their mark in themselves, so NFD leaves them whole: "Włochy", "Færøerne", "Türkiye" typed
// with a dotless ı.
const FOLDS: Record<string, string> = {
    "ł": "l", "ø": "o", "đ": "d", "ð": "d", "ı": "i", "ħ": "h", "ŧ": "t",
    "æ": "ae", "œ": "oe", "ß": "ss", "þ": "th",
    "’": "'", "‘": "'", "ʼ": "'",
}

// The name as a player types it: no case, no accent, a plain apostrophe. "Népal" and "Côte d’Ivoire" are
// "nepal" and "cote d'ivoire".
export function foldForSearch(text: string, locale: string): string {
    return text.toLocaleLowerCase(locale)
        .normalize("NFD")
        .replace(/\p{M}/gu, "")
        .replace(/[łøđðıħŧæœßþ’‘ʼ]/g, (c) => FOLDS[c])
}

export function matchesSearch(value: Value, search: string): boolean {
    const locale = translator().locale
    const term = foldForSearch(search.trim(), locale)
    return term === "" || foldForSearch(value.name, locale).includes(term)
}

export function visibleOptions(values: Value[], selected: Value | undefined, search: string): Value[] {
    const matches = values.filter((v) => matchesSearch(v, search))

    if (selected === undefined || matches.some((v) => v.code === selected.code)) return matches
    return [selected, ...matches]
}
