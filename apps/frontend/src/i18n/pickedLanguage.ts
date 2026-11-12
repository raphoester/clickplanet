import {Language} from "./languages.ts"

const LANGUAGE_STORAGE_KEY = "clickplanet-language"

export function readPickedLanguage(): string | null {
    try {
        return window.localStorage.getItem(LANGUAGE_STORAGE_KEY)
    } catch {
        return null
    }
}

export function keepPickedLanguage(language: Language) {
    try {
        window.localStorage.setItem(LANGUAGE_STORAGE_KEY, language)
    } catch (e) {
        console.error("Could not keep the language picked", e)
    }
}
