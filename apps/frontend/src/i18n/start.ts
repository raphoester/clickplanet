import {chooseLanguage} from "./languages.ts"
import {translatorFor} from "./load.ts"
import {speak} from "./page.ts"
import {readPickedLanguage} from "./pickedLanguage.ts"
import {ENGLISH_TRANSLATOR, Translator} from "./translator.ts"

// CrazyGames' SDK may never load: its locale is waited for this long, then the browser's languages decide.
const PORTAL_LOCALE_WAIT_MS = 2_000

// The page's language, before it draws: the pick kept, then the portal's, then the browser's. A file that does not
// load leaves the page in English.
export async function speakChosenLanguage(portal?: Promise<string | undefined>): Promise<Translator> {
    const browser = navigator.languages ?? [navigator.language]
    const language = chooseLanguage({picked: readPickedLanguage(), portal: portal && await soonOrNone(portal), browser})
    const translator = await translatorFor(language, browser).catch((e) => {
        console.error(`The ${language} lines did not load: the page speaks English`, e)
        return ENGLISH_TRANSLATOR
    })
    speak(translator)
    document.documentElement.lang = translator.language
    return translator
}

function soonOrNone(portal: Promise<string | undefined>): Promise<string | undefined> {
    return Promise.race([portal, new Promise<undefined>((resolve) => setTimeout(resolve, PORTAL_LOCALE_WAIT_MS))])
}
