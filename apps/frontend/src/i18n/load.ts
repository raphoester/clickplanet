import type {BoardAsset} from "../domain/board.ts"
import {Language, SOURCE_LANGUAGE} from "./languages.ts"
import {ENGLISH, ENGLISH_COUNTRIES, LandmassNames, Lines, NO_PROVINCE_NAMES, ProvinceNames, Translator} from "./translator.ts"

const FILES = import.meta.glob<{default: Lines}>(["./locales/*.json", "!./locales/en.json"])
const LANDMASS_FILES = import.meta.glob<{default: Lines}>("./landmasses/*/*.json")
const PROVINCE_FILES = import.meta.glob<{default: Lines}>("./provinces/*/*.json")
const COUNTRY_FILES = import.meta.glob<{default: Lines}>(["./countries/*.json", "!./countries/en.json"])

// Only the player's language is downloaded: English is in the bundle, and fills any line a file lacks.
export async function translatorFor(language: Language, browser: readonly string[]): Promise<Translator> {
    if (language === SOURCE_LANGUAGE) return new Translator(language, ENGLISH, localeOf(language, browser), ENGLISH_COUNTRIES)
    const [lines, countries] = await Promise.all([
        FILES[`./locales/${language}.json`](),
        COUNTRY_FILES[`./countries/${language}.json`](),
    ])
    return new Translator(language, lines.default, localeOf(language, browser), countries.default)
}

// A board's landmass names: English from its own file, served beside it, and the language's from the catalogue, keyed
// by the board, since a landmass id is a row of one board's borders file. A board with none in the language reads
// English.
export async function landmassNamesOf(board: BoardAsset, language: Language, signal?: AbortSignal): Promise<LandmassNames> {
    const translated = language === SOURCE_LANGUAGE ? undefined : LANDMASS_FILES[`./landmasses/${board.name}/${language}.json`]
    const [english, own] = await Promise.all([
        fetch(board.urls.landmassNames, {signal}).then(async (answer) => {
            if (!answer.ok) throw new Error(`Failed to fetch ${board.urls.landmassNames}: ${answer.status} ${answer.statusText}`)
            return ((await answer.json()) as {names: Lines}).names
        }),
        translated ? translated().then((file) => file.default) : {},
    ])
    return {english, own}
}

// A board's province names, as its landmass names: English beside the board, the language's from the catalogue. A board
// with no provinces has none.
export async function provinceNamesOf(board: BoardAsset, language: Language, signal?: AbortSignal): Promise<ProvinceNames> {
    const url = board.urls.provinceNames
    if (url === undefined) return NO_PROVINCE_NAMES
    const translated = language === SOURCE_LANGUAGE ? undefined : PROVINCE_FILES[`./provinces/${board.name}/${language}.json`]
    const [english, own] = await Promise.all([
        fetch(url, {signal}).then(async (answer) => {
            if (!answer.ok) throw new Error(`Failed to fetch ${url}: ${answer.status} ${answer.statusText}`)
            return ((await answer.json()) as {names: Lines}).names
        }),
        translated ? translated().then((file) => file.default) : {},
    ])
    return {english, own}
}

// "en-GB" for English when the browser says so: its dates and digits, in the language chosen.
export function localeOf(language: Language, browser: readonly string[]): string {
    const primary = new Intl.Locale(language).language
    for (const tag of browser) {
        try {
            const locale = new Intl.Locale(tag)
            if (locale.language === primary) return locale.toString()
        } catch {
            // A tag Intl does not take says nothing.
        }
    }
    return language
}
