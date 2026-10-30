import {EVERY_WORD_MASKED, WordFilter, wordFilterOf} from "../domain/wordFilter.ts"

export const CRAZYGAMES_WORDS_URL = "/crazygames/words.txt"

export async function crazyGamesWords(
    url: string,
    fetch: typeof globalThis.fetch = (input, init) => globalThis.fetch(input, init),
): Promise<WordFilter> {
    try {
        const response = await fetch(url)
        // A missing file is the home page, answered 200 by the single-page fallback.
        const type = response.headers.get("content-type") ?? ""
        if (!response.ok || !type.startsWith("text/plain")) throw new Error(`${url} answered ${response.status} ${type}`)
        return wordFilterOf((await response.text()).split("\n"))
    } catch (e) {
        console.error("the CrazyGames word list did not load: every word is masked", e)
        return EVERY_WORD_MASKED
    }
}
