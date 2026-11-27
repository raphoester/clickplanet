import {loadDisplayFace} from "./displayFace.ts"
import {chooseLanguage, directionOf} from "./languages.ts"
import {translatorFor} from "./load.ts"
import {speak} from "./page.ts"
import {readPickedLanguage} from "./pickedLanguage.ts"
import {ENGLISH_TRANSLATOR, Translator} from "./translator.ts"

// The page's language, before it draws: the pick kept, then the browser's. A file that does not load leaves the page
// in English.
export async function speakChosenLanguage(): Promise<Translator> {
    const browser = navigator.languages ?? [navigator.language]
    const language = chooseLanguage({picked: readPickedLanguage(), browser})
    const translator = await translatorFor(language, browser).catch((e) => {
        console.error(`The ${language} lines did not load: the page speaks English`, e)
        return ENGLISH_TRANSLATOR
    })
    speak(translator)
    document.documentElement.lang = translator.language
    document.documentElement.dir = directionOf(translator.language)
    loadDisplayFace(translator.language)
    return translator
}
