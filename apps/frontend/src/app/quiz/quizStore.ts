import {create} from 'zustand'
import {BonusLostError, QuizMaster} from '../../backends/backend.ts'
import {now as budgetNow} from '../../backends/clickBudget.ts'
import {QuizOffer, QuizOutcome, QuizQuestion} from '../../domain/quiz.ts'
import {PlaySound} from '../sound/soundPlayer.ts'

export const RESULT_MS = 3200

type QuizState =
    | {phase: 'idle'}
    | {phase: 'offered', offer: QuizOffer}
    | {phase: 'opening', offer: QuizOffer}
    | {phase: 'asking', token: string, question: QuizQuestion}
    | {phase: 'answering', token: string, question: QuizQuestion}
    | {
        phase: 'answered',
        question: QuizQuestion,
        outcome: QuizOutcome,
        chosen?: number,
        left?: boolean,
    }

type QuizLink = {
    master: QuizMaster
    countryCode: string
    play: PlaySound
}

type QuizStore = {
    state: QuizState
    link?: QuizLink
    offer: (offer: QuizOffer) => void
    open: () => void
    answer: (choice: number) => void
    forfeit: () => void
    dismiss: () => void
}

const IDLE: QuizState = {phase: 'idle'}

export const quizShown = (quiz: QuizStore) => quiz.state.phase !== 'idle'

export const useQuiz = create<QuizStore>()((set, get) => {
    const leave = (shown: QuizState) => set((quiz) => quiz.state === shown ? {state: IDLE} : quiz)

    const send = (choice: number, left: boolean) => {
        const {state, link} = get()
        if (!link || state.phase !== 'asking') return

        const {token, question} = state
        const chosen = choice < question.choices.length ? choice : undefined
        set({state: {phase: 'answering', token, question}})

        link.master.answerQuiz(token, choice, link.countryCode)
            .then((outcome) => {
                const answered: QuizState = {phase: 'answered', question, outcome, chosen, left}
                set({state: answered})
                get().link?.play(outcome.correct ? 'quizRight' : 'quizWrong')
                const clear = () => setTimeout(() => leave(answered), RESULT_MS)
                if (left) whenBack(clear)
                else clear()
            })
            .catch((error) => {
                if (!(error instanceof BonusLostError)) console.error('could not answer the quiz', error)
                set({state: IDLE})
            })
    }

    return {
        state: IDLE,
        offer: (offer) => {
            if (get().state.phase !== 'idle') return

            const offered: QuizState = {phase: 'offered', offer}
            set({state: offered})
            get().link?.play('quiz')
            setTimeout(() => leave(offered), Math.max(0, offer.expiresAt - budgetNow()))
        },
        open: () => {
            const {state, link} = get()
            if (!link || state.phase !== 'offered') return

            const {token} = state.offer
            const opening: QuizState = {phase: 'opening', offer: state.offer}
            set({state: opening})

            link.master.openQuiz(token)
                .then((question) => set((quiz) => quiz.state === opening ? {state: {phase: 'asking', token, question}} : quiz))
                .catch((error) => {
                    if (!(error instanceof BonusLostError)) console.error('could not open the quiz', error)
                    leave(opening)
                })
        },
        answer: (choice) => send(choice, false),
        forfeit: () => {
            const {state} = get()
            if (state.phase === 'asking') send(state.question.choices.length, true)
        },
        dismiss: () => {
            const {state} = get()
            if (state.phase === 'answered') leave(state)
        },
    }
})

// Calls back at once when the page has the focus, or else when it next gets it or is pressed: a result that says the
// quiz was left must still be there when the player comes back to read it.
function whenBack(callback: () => void) {
    if (document.hasFocus()) {
        callback()
        return
    }
    const back = () => {
        window.removeEventListener('focus', back)
        window.removeEventListener('pointerdown', back, true)
        callback()
    }
    window.addEventListener('focus', back)
    window.addEventListener('pointerdown', back, true)
}
