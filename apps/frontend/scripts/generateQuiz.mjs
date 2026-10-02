// Writes the quiz bank: every question the game can ask, and the answer to each.
//
//     npm run quiz:generate            # rewrites /quiz/bank-<hash>.json
//     cd apps/backend && make quiz     # copies it in, to be embedded
//
// **The bank is the backend's alone.** It is the only generated thing in this repo that does not
// get a copy under `apps/frontend/static`, because the answers are in it: a browser that can fetch
// the bank can win every quiz, and a file served to the page is a file anyone can fetch. The
// question and its three choices reach the client one at a time, from `OpenQuiz`, and which of the
// three is right never leaves the server at all.
//
// Two sources, and the split between them is the point:
//
//   - **Derived** — capitals, borders, continents and populations, from the same pinned Natural
//     Earth snapshot the map is cut from and from the tile borders themselves. About a thousand
//     questions that are regenerated rather than corrected.
//   - **Hand-written** — `scripts/quiz/extra.json`, for the questions no dataset here can answer.
//     Kept to facts that do not move, or dated so they cannot; see the note at the top of that file.
//
// See /quiz/README.md.
import crypto from "node:crypto"
import fs from "node:fs"
import path from "node:path"
import {fileURLToPath} from "node:url"

import {NATURAL_EARTH_TAG} from "./map/naturalEarth.mjs"
import {countryFacts, gameNames} from "./quiz/facts.mjs"
import {DISTRACTORS, questionsFrom} from "./quiz/templates.mjs"

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

const questions = [...derived, ...extra]
check(questions)

const bank = {
    format: 1,
    naturalEarthTag: NATURAL_EARTH_TAG,
    // No timestamp: the name is the content hash, and a clock in the body would give the same
    // questions a different name every run.
    questions,
}

write(bank)

// --- the hand-written half.

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

// --- what a bank has to be true of before it is written.

function check(questions) {
    const ids = new Set()

    for (const question of questions) {
        const where = question.id ?? question.text

        if (!question.id || ids.has(question.id)) throw new Error(`${where}: id is missing or repeated`)
        ids.add(question.id)

        if (!question.text || !question.answer) throw new Error(`${where}: no question, or no answer`)

        // Two of the wrong answers are shown beside the right one, so two is the floor. The ceiling
        // is only what the templates promise; a hand-written question with more is a typo.
        if (question.wrong.length < 2 || question.wrong.length > DISTRACTORS) {
            throw new Error(`${where}: ${question.wrong.length} wrong answers, want 2 to ${DISTRACTORS}`)
        }

        // A right answer that is also one of the wrong ones is a question with no answer, and it is
        // the one fault a generated bank can produce quietly: two countries share a capital's name,
        // or a distractor drawn from the same continent turns out to be the neighbour.
        const wrong = new Set(question.wrong)
        if (wrong.size !== question.wrong.length) throw new Error(`${where}: a wrong answer is repeated`)
        if (wrong.has(question.answer)) throw new Error(`${where}: "${question.answer}" is both right and wrong`)
    }
}

// --- writing it.

function write(bank) {
    const body = `${JSON.stringify(bank, null, 2)}\n`
    const hash = crypto.createHash("sha256").update(body).digest("hex").slice(0, 8)
    const name = `bank-${hash}.json`

    fs.mkdirSync(quizDir, {recursive: true})

    // The name is the content hash, so the old one has to go: the backend embeds `bank-*.json` and
    // a second match is a build that cannot pick.
    for (const stale of fs.readdirSync(quizDir).filter((entry) => /^bank-[0-9a-f]{8}\.json$/.test(entry))) {
        if (stale !== name) fs.rmSync(path.join(quizDir, stale))
    }

    fs.writeFileSync(path.join(quizDir, name), body)

    say("")
    say(`wrote quiz/${name}: ${bank.questions.length} questions, ${(body.length / 1024).toFixed(0)} KiB`)
    say("")
    say("next:")
    say("  cd apps/backend && make quiz   # copy it in to be embedded, then commit both")
}
