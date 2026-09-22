/**
 * A quiz: three choices, five seconds, and a charge for the right answer.
 *
 * **The client never knows the answer before it is given.** The bank lives on the server and is
 * deliberately not shipped to the browser (see /quiz/README.md); the question and its three choices
 * arrive one at a time, and which of the three is right comes back only in the answer. So nothing
 * in this file decides anything — it is the shape the server's answers arrive in, and the two small
 * pieces of arithmetic a countdown needs.
 */
import {BonusReward} from "./bonus.ts"

/**
 * The banner: a quiz the server has put in front of **this** client, and nobody else.
 *
 * **It says nothing about the question** — not the text, not the choices, and not even what it is
 * about. A banner is only an invitation, and anything written on one is something a player can
 * read while the clock is not running. The five seconds start when the question is read.
 *
 * It used to name the subject country so the banner could fly a flag. That flag *was* the answer
 * to 417 of the bank's 1014 questions: every "Tallinn is the capital of which country?" and every
 * "which of these has the most people?". A teaser that has to be checked against every question in
 * the bank leaks again the first time a template is added, so there is none.
 */
export type QuizOffer = {
    /** Single use, and worth nothing to any other caller. */
    token: string

    /**
     * When the banner stops being clickable, on the monotonic clock — the same one `ClickBudget` is
     * stamped against.
     *
     * Derived from how long the server said was *left*, not from the timestamp it sent: the two
     * machines' wall clocks are unrelated, and a client whose clock is a minute out would otherwise
     * think every banner had already lapsed.
     */
    expiresAt: number
}

/** The question, once it is opened. The clock is running from here. */
export type QuizQuestion = {
    text: string

    /** Three of them, in the order to show. Which is right is not here. */
    choices: string[]

    /** When an answer stops counting, on the monotonic clock, derived as `QuizOffer.expiresAt` is. */
    deadline: number

    /**
     * The whole window the player was given, in milliseconds. The countdown is drawn against this
     * rather than against what a slow round trip left of it, so the bar always starts full.
     */
    window: number
}

/** What an answer was worth. */
export type QuizOutcome = {
    correct: boolean

    /** Which choice was right, whatever was pressed: a quiz nobody learns from is a worse quiz. */
    correctChoice: number

    /** What the right answer won. Undefined when it was wrong, and undefined for a kind this build does not know. */
    reward?: BonusReward
}

/**
 * How much of the window is left, 0 to 1, for the bar. Clamped at both ends: a deadline already
 * past is an empty bar rather than a negative one, and a clock that ran backwards is a full one.
 */
export function timeLeft(question: QuizQuestion, at: number): number {
    if (question.window <= 0) return 0
    return Math.max(0, Math.min(1, (question.deadline - at) / question.window))
}

/** Whether the question can still be answered. The server checks this too, and its answer is the one that counts. */
export function stillOpen(question: QuizQuestion, at: number): boolean {
    return at < question.deadline
}
