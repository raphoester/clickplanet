import {Language, SOURCE_LANGUAGE} from "./languages.ts"
import {ENGLISH, ENGLISH_COUNTRIES, ENGLISH_LANDMASSES, Lines, Translator} from "./translator.ts"

const FILES = import.meta.glob<{default: Lines}>(["./locales/*.json", "!./locales/en.json"])
const LANDMASS_FILES = import.meta.glob<{default: Lines}>("./landmasses/*.json")
const COUNTRY_FILES = import.meta.glob<{default: Lines}>(["./countries/*.json", "!./countries/en.json"])

// Only the player's language is downloaded: English is in the bundle, and fills any line a file lacks.
export async function translatorFor(language: Language, browser: readonly string[]): Promise<Translator> {
    if (language === SOURCE_LANGUAGE) return new Translator(language, ENGLISH, localeOf(language, browser), ENGLISH_LANDMASSES, ENGLISH_COUNTRIES)
    const [lines, landmasses, countries] = await Promise.all([
        FILES[`./locales/${language}.json`](),
        LANDMASS_FILES[`./landmasses/${language}.json`](),
        COUNTRY_FILES[`./countries/${language}.json`](),
    ])
    return new Translator(language, lines.default, localeOf(language, browser), landmasses.default, countries.default)
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
