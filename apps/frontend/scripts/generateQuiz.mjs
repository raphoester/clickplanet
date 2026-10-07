import fs from "node:fs"
import path from "node:path"
import {fileURLToPath} from "node:url"

import {NATURAL_EARTH_TAG} from "./map/naturalEarth.mjs"
import {countryFacts, gameNames} from "./quiz/facts.mjs"
import {DISTRACTORS, questionsFrom} from "./quiz/templates.mjs"
import {names} from "./quiz/words.mjs"

const frontendRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..")
const quizDir = path.resolve(frontendRoot, "..", "..", "quiz")
const extraFile = path.join(frontendRoot, "scripts", "quiz", "extra.json")

const say = (...args) => console.error(...args)

const facts = await countryFacts()
say(`${facts.size} countries the game has a name for`)

const derived = questionsFrom(facts)
say(`${derived.length} questions derived from Natural Earth ${NATURAL_EARTH_TAG} and the tile borders`)

const extra = handWritten()
say(`${extra.length} questions written by hand`)

const countries = gameNames()
const questions = [...derived, ...extra].map((question) => ({
    ...question,
    namesSubject: question.subject !== "" && names(question.text, countries.get(question.subject)),
}))
check(questions)

const bank = {
    format: 1,
    naturalEarthTag: NATURAL_EARTH_TAG,
    // No timestamp: a run that asks the same questions writes the same file.
    questions,
}

write(bank)

function handWritten() {
    const names = gameNames()
    const {questions: written} = JSON.parse(fs.readFileSync(extraFile, "utf8"))

    return written.map((question) => {
        if (question.subject && !names.has(question.subject)) {
            throw new Error(`${question.id}: subject "${question.subject}" is not a country the game knows`)
        }

        return {
            id: question.id,
            subject: question.subject ?? "",
            ask: "extra",
            text: question.text,
            answer: question.answer,
            wrong: question.wrong,
        }
    })
}

function check(questions) {
    const ids = new Set()

    for (const question of questions) {
        const where = question.id ?? question.text

        if (!question.id || ids.has(question.id)) throw new Error(`${where}: id is missing or repeated`)
        ids.add(question.id)

        if (!question.text || !question.answer) throw new Error(`${where}: no question, or no answer`)

        if (question.wrong.length < 2 || question.wrong.length > DISTRACTORS) {
            throw new Error(`${where}: ${question.wrong.length} wrong answers, want 2 to ${DISTRACTORS}`)
        }

        const wrong = new Set(question.wrong)
        if (wrong.size !== question.wrong.length) throw new Error(`${where}: a wrong answer is repeated`)
        if (wrong.has(question.answer)) throw new Error(`${where}: "${question.answer}" is both right and wrong`)
    }
}

// One question per line, so a diff of the bank is one line per question that changed.
function serialized(bank) {
    const questions = bank.questions.map((question) => `    ${JSON.stringify(question)}`).join(",\n")

    return [
        "{",
        `  "format": ${JSON.stringify(bank.format)},`,
        `  "naturalEarthTag": ${JSON.stringify(bank.naturalEarthTag)},`,
        `  "questions": [`,
        questions,
        "  ]",
        "}",
        "",
    ].join("\n")
}

function write(bank) {
    const body = serialized(bank)

    fs.mkdirSync(quizDir, {recursive: true})
    fs.writeFileSync(path.join(quizDir, "bank.json"), body)

    say("")
    say(`wrote quiz/bank.json: ${bank.questions.length} questions, ${(body.length / 1024).toFixed(0)} KiB`)
    say("")
    say("next:")
    say("  cd apps/backend && make quiz   # copy it in to be embedded, then commit both")
}
