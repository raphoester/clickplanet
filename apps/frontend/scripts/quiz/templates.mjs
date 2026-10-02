import {first, pick, shuffled} from "./random.mjs"

export const DISTRACTORS = 5

/**
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

function near(country, all) {
    const others = all.filter((other) => other.code !== country.code)
    const rank = (other) => {
        if (country.subregion && other.subregion === country.subregion) return 0
        if (country.continent && other.continent === country.continent) return 1
        return 2
    }

    return shuffled(others, `near:${country.code}`).sort((a, b) => rank(a) - rank(b))
}
