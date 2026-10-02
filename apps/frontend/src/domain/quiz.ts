import {BonusReward} from "./bonus.ts"

// Nothing about the question: a banner can be read before the answer clock starts.
export type QuizOffer = {
    token: string

    expiresAt: number
}

export type QuizQuestion = {
    text: string

    choices: string[]

    deadline: number

    window: number
}

export type QuizOutcome = {
    correct: boolean

    correctChoice: number

    reward?: BonusReward
}

export function timeLeft(question: QuizQuestion, at: number): number {
    if (question.window <= 0) return 0
    return Math.max(0, Math.min(1, (question.deadline - at) / question.window))
}

export function stillOpen(question: QuizQuestion, at: number): boolean {
    return at < question.deadline
}
