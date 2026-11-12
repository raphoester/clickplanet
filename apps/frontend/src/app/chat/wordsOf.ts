import {Gif} from "../../backends/chat.ts"
import {t} from "../../i18n/page.ts"

// What a line that has room for words alone says of a message or a quote.
export function wordsOf(said: {text: string, gif?: Gif}): string {
    return said.gif ? t("chat.gif") : said.text
}
