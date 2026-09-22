// The question templates, and the rule each one is true by.
//
// A template is a fact the generator can *check*, not a fact it was told. Every question below is
// derived from the same Natural Earth snapshot the map is cut from, or from the tile borders
// themselves, so the bank is regenerated rather than corrected: no answer here can rot on its own
// while the data it came from stays pinned.
//
// Each question carries more wrong answers than a round shows. The server picks two of them when
// the player opens the quiz, so the same question is not the same three choices twice, and a crib
// sheet of "press the second one for Estonia" is worth nothing.
import {first, pick, shuffled} from "./random.mjs"

/** How many wrong answers a question carries. The round shows two of them. */
export const DISTRACTORS = 5

/**
 * Every question the facts support, in a fixed order.
 *
 * @param {Map<string, import("./facts.mjs").Facts>} facts
 * @returns {{id: string, subject: string, ask: string, text: string, answer: string, wrong: string[]}[]}
 */
export function questionsFrom(facts) {
    const all = [...facts.values()].sort((a, b) => (a.code < b.code ? -1 : 1))

    return [
        ...all.flatMap((country) => capital(country, all)),
        ...all.flatMap((country) => capitalOf(country, all)),
        ...all.flatMap((country) => borders(country, facts, all)),
        ...all.flatMap((country) => continent(country, all)),
        ...all.flatMap((country) => mostPeople(country, all)),
    ]
}

// --- the templates.

// "What is the capital of Estonia?" — wrong answers from as close as possible, because Riga and
// Vilnius are a question and Canberra and Lima are a giveaway.
function capital(country, all) {
    if (!country.capital) return []

    const wrong = first(near(country, all).map((other) => other.capital), DISTRACTORS)
    if (wrong.length < 2) return []

    return [{
        id: `capital:${country.code}`,
        subject: country.code,
        ask: "capital",
        text: `What is the capital of ${country.name}?`,
        answer: country.capital,
        wrong,
    }]
}

// The same fact asked the other way round, which is a different question to answer in five seconds.
function capitalOf(country, all) {
    if (!country.capital) return []

    const wrong = first(near(country, all).map((other) => other.name), DISTRACTORS)
    if (wrong.length < 2) return []

    return [{
        id: `capitalOf:${country.code}`,
        subject: country.code,
        ask: "capitalOf",
        text: `${country.capital} is the capital of which country?`,
        answer: country.name,
        wrong,
    }]
}

// "Which of these shares a border with Nepal?" — the border is the one on the map being played,
// and so are the wrong answers: a country on the same continent that no tile of this one touches.
function borders(country, facts, all) {
    if (country.neighbours.length === 0) return []

    const answer = facts.get(pick(country.neighbours, 1, `borders:${country.code}`)[0])
    if (!answer) return []

    const strangers = all.filter((other) =>
        other.code !== country.code
        && other.continent === country.continent
        && !country.neighbours.includes(other.code))

    const wrong = pick(strangers.map((other) => other.name), DISTRACTORS, `borders:wrong:${country.code}`)
    if (wrong.length < 2) return []

    return [{
        id: `borders:${country.code}`,
        subject: country.code,
        ask: "borders",
        text: `Which of these shares a border with ${country.name}?`,
        answer: answer.name,
        wrong,
    }]
}

function continent(country, all) {
    if (!country.continent) return []

    const others = [...new Set(all.map((other) => other.continent).filter(Boolean))]
        .filter((name) => name !== country.continent)

    const wrong = pick(others, DISTRACTORS, `continent:${country.code}`)
    if (wrong.length < 2) return []

    return [{
        id: `continent:${country.code}`,
        subject: country.code,
        ask: "continent",
        text: `Which continent is ${country.name} in?`,
        answer: country.continent,
        wrong,
    }]
}

// "Which of these has the most people?", asked about the country that has them.
//
// A population is the one fact here that moves under a pinned snapshot, so the question is only
// built where the answer wins by MARGIN times over — an ordering that survives a decade of either
// country growing. It is why the wrong answers are drawn from the *smaller* half and not from the
// continent at large.
const MARGIN = 2

function mostPeople(country, all) {
    if (country.population <= 0) return []

    const smaller = all.filter((other) =>
        other.code !== country.code
        && other.continent === country.continent
        && other.population > 0
        && country.population >= other.population * MARGIN)

    const wrong = pick(smaller.map((other) => other.name), DISTRACTORS, `mostPeople:${country.code}`)
    if (wrong.length < 2) return []

    return [{
        id: `mostPeople:${country.code}`,
        subject: country.code,
        ask: "mostPeople",
        text: "Which of these countries has the most people?",
        answer: country.name,
        wrong,
    }]
}

// Countries whose capitals make a hard question of this one's: its own subregion first, then its
// continent, then the world. Sorted before the draw, so the bank is the same bytes every run.
function near(country, all) {
    const others = all.filter((other) => other.code !== country.code)
    const rank = (other) => {
        if (country.subregion && other.subregion === country.subregion) return 0
        if (country.continent && other.continent === country.continent) return 1
        return 2
    }

    return shuffled(others, `near:${country.code}`).sort((a, b) => rank(a) - rank(b))
}
