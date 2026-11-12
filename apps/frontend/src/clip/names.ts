import gameNames from "../../static/countries/countries.json"

const GAME_NAMES: Readonly<Record<string, string>> = gameNames

const ENGLISH = new Intl.DisplayNames(["en"], {type: "region", style: "short"})

// The game's names are cut to fit its board ("Christmas", "N.Zealand", "Czech Rep."): a clip says the English name they
// were cut from, from the CLDR data the browser carries. A name the game chose over it stays: "Turkey", "Ivory Coast",
// "UAE", and its own flags'.
export function nameOf(code: string): string {
    const game = GAME_NAMES[code] ?? code.toUpperCase()
    const english = englishNameOf(code)
    return english !== undefined && cutFrom(game, english) ? english : game
}

// ISO's x codes are for private use: CLDR has a test name where the game has Brittany.
function englishNameOf(code: string): string | undefined {
    if (!/^[a-wyz][a-z]$/.test(code)) return undefined
    const name = ENGLISH.of(code.toUpperCase())
    // "Congo - Kinshasa" names two places to tell them apart; the game's "DR Congo" says it better.
    return name === undefined || name.includes(" - ") ? undefined : name.replace(/ \(.*\)$/, "")
}

function cutFrom(game: string, english: string): boolean {
    if (/[.,]/.test(game)) return true
    const wordsOf = (name: string) => name.normalize("NFD").replace(/\p{M}/gu, "").toLowerCase().split(/[^a-z]+/).filter(Boolean)
    const full = wordsOf(english)
    return english.length > game.length && wordsOf(game).every((word) => full.some((whole) => whole.startsWith(word)))
}

// The UK, the Netherlands: a name said with its article.
const SAID_WITH_THE = /^(UK|USA|UAE|Gambia|Middle East)$|(lands|ines|amas|ives|oros|elles|Republic|Territories)$/

export function said(name: string): string {
    return SAID_WITH_THE.test(name) ? `the ${name}` : name
}

// Luckiest Guy draws Latin-1 and little more: "ŁOTWA" in it would mix two fonts in one word.
export const DISPLAY_FONT_DRAWS = /^[\u0020-\u00ff\u0131\u0152\u0153\u2000-\u206f\u20ac\u2122]*$/u
