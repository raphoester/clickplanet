import {ENGLISH_TRANSLATOR, Key, Translator, Values} from "./translator.ts"

let current: Translator = ENGLISH_TRANSLATOR

// The page speaks one language, chosen before it draws: a pick in the settings reloads it.
export function speak(translator: Translator) {
    current = translator
}

export function translator(): Translator {
    return current
}

export function t(key: Key, values?: Values): string {
    return current.t(key, values)
}
