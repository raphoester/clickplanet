export const LANGUAGES = ["en", "fr", "de", "tr", "pt-BR", "pl", "id", "es", "cs"] as const

export type Language = typeof LANGUAGES[number]

export const SOURCE_LANGUAGE: Language = "en"

// Slovak players read Czech; every Portuguese and Spanish reads the one we have.
const BY_PRIMARY: Readonly<Record<string, Language>> = {
    en: "en",
    fr: "fr",
    de: "de",
    tr: "tr",
    pt: "pt-BR",
    pl: "pl",
    id: "id",
    es: "es",
    cs: "cs",
    sk: "cs",
}

export function languageOf(tag: string | null | undefined): Language | undefined {
    if (!tag) return undefined
    let primary: string
    try {
        primary = new Intl.Locale(tag).language
    } catch {
        return undefined
    }
    return BY_PRIMARY[primary]
}

export function isLanguage(value: string | null): value is Language {
    return (LANGUAGES as readonly (string | null)[]).includes(value)
}

export type LanguageHints = {
    // The player's own pick in the settings.
    picked: string | null
    // CrazyGames' locale, on its page.
    portal?: string
    // navigator.languages, the player's order.
    browser: readonly string[]
}

export function chooseLanguage({picked, portal, browser}: LanguageHints): Language {
    if (isLanguage(picked)) return picked
    const fromPortal = languageOf(portal)
    if (fromPortal) return fromPortal
    for (const tag of browser) {
        const language = languageOf(tag)
        if (language) return language
    }
    return SOURCE_LANGUAGE
}

// A language's own name, as its speakers write it: "Deutsch", "Čeština".
export function nativeNameOf(language: Language): string {
    const name = new Intl.DisplayNames([language], {type: "language"}).of(language) ?? language
    return name.charAt(0).toLocaleUpperCase(language) + name.slice(1)
}
