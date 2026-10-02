import {useCallback, useEffect, useRef, useState} from 'react'
import {BonusLostError, QuizMaster} from '../../backends/backend.ts'
import {now as budgetNow} from '../../backends/clickBudget.ts'
import {QuizOffer, QuizOutcome, QuizQuestion} from '../../domain/quiz.ts'
import {PlaySound} from '../sound/soundPlayer.ts'

export const RESULT_MS = 3200

export type QuizState =
    | {phase: 'idle'}
    | {phase: 'offered', offer: QuizOffer}
    | {phase: 'opening', offer: QuizOffer}
    | {phase: 'asking', token: string, question: QuizQuestion}
    | {
        phase: 'answered',
        question: QuizQuestion,
        outcome: QuizOutcome,
        chosen?: number,
    }

export function useQuiz(master?: QuizMaster, countryCode?: string, playSound?: PlaySound) {
    const [state, setState] = useState<QuizState>({phase: 'idle'})

    const play = useRef(playSound)
    useEffect(() => {
        play.current = playSound
    }, [playSound])

    const country = useRef(countryCode)
    useEffect(() => {
        country.current = countryCode
    }, [countryCode])

    const current = useRef(state)
    useEffect(() => {
        current.current = state
    }, [state])

    const dismiss = useCallback(() => setState({phase: 'idle'}), [])

    useEffect(() => {
        if (!master) return

        return master.listenForQuizzes((offer) => {
            if (current.current.phase !== 'idle') return
            setState({phase: 'offered', offer})
            play.current?.("quiz")
        })
    }, [master])

    useEffect(() => {
        if (state.phase !== 'offered') return

        const left = state.offer.expiresAt - budgetNow()
        if (left <= 0) {
            dismiss()
            return
        }

        const timer = setTimeout(dismiss, left)
        return () => clearTimeout(timer)
    }, [state, dismiss])

    useEffect(() => {
        if (state.phase !== 'answered') return

        const timer = setTimeout(dismiss, RESULT_MS)
        return () => clearTimeout(timer)
    }, [state, dismiss])

    const open = useCallback(() => {
        const showing = current.current
        if (!master || showing.phase !== 'offered') return

        const {offer} = showing
        current.current = {phase: 'opening', offer}
        setState(current.current)

        master.openQuiz(offer.token)
            .then((question) => setState((now) =>
                now.phase === 'opening' && now.offer.token === offer.token
                    ? {phase: 'asking', token: offer.token, question}
                    : now))
            .catch((error) => {
                if (!(error instanceof BonusLostError)) console.error("could not open the quiz", error)
                setState((now) => now.phase === 'opening' && now.offer.token === offer.token ? {phase: 'idle'} : now)
            })
    }, [master])

    const answer = useCallback((choice: number) => {
        const showing = current.current
        if (!master || showing.phase !== 'asking') return

        const {token, question} = showing
        const chosen = choice < question.choices.length ? choice : undefined

        // Leaves 'asking' at once, so a second press or the timeout sends nothing.
        current.current = {phase: 'opening', offer: {token, expiresAt: question.deadline}}

        master.answerQuiz(token, choice, country.current ?? "")
            .then((outcome) => {
                setState({phase: 'answered', question, outcome, chosen})
                play.current?.(outcome.correct ? "quizRight" : "quizWrong")
            })
            .catch((error) => {
                if (!(error instanceof BonusLostError)) console.error("could not answer the quiz", error)
                setState({phase: 'idle'})
            })
    }, [master])

    return {state, open, answer, dismiss}
}
