import {useCallback, useEffect, useRef, useState} from 'react'
import {BonusLostError, QuizMaster} from '../../backends/backend.ts'
import {now as budgetNow} from '../../backends/clickBudget.ts'
import {QuizOffer, QuizOutcome, QuizQuestion} from '../../domain/quiz.ts'
import {PlaySound} from '../sound/soundPlayer.ts'

/**
 * How long the result stays up once an answer has landed. Long enough to read the right answer
 * when it was not the one pressed, which is the only teaching this feature does.
 */
export const RESULT_MS = 3200

/**
 * The one quiz there is at a time, from the banner appearing to the result going away.
 *
 *     idle → offered → opening → asking → answered → idle
 *
 * The phases exist because each is a different thing on screen *and* a different thing to a
 * player: `offered` is an invitation that costs nothing to ignore, and `asking` is five seconds
 * that are already running. Anything that goes wrong — a token the server will not honour, a
 * stream that dropped — falls back to `idle`: a quiz nobody can answer should leave nothing behind.
 *
 * The token rides all the way through rather than being kept beside the state, so there is no way
 * to answer one quiz with another's token while a banner and a question are changing places.
 */
export type QuizState =
    | {phase: 'idle'}
    | {phase: 'offered', offer: QuizOffer}
    /** Opened, and waiting for the question. A second press must not open it twice. */
    | {phase: 'opening', offer: QuizOffer}
    | {phase: 'asking', token: string, question: QuizQuestion}
    | {
        phase: 'answered',
        question: QuizQuestion,
        outcome: QuizOutcome,
        /** Which one was pressed, or undefined when the time ran out with nothing pressed. */
        chosen?: number,
    }

/**
 * Runs that state machine against a backend.
 *
 * **It is React's own, and never touches the globe.** A quiz is DOM at the top of the screen, not
 * an object in the scene, so it subscribes to the feed itself rather than being handed down
 * through `globe.ts` the way a flying box is. What a right answer wins reaches the inventory the
 * way every other charge does — the backend holds the charges and tells whoever is listening.
 */
export function useQuiz(master?: QuizMaster, countryCode?: string, playSound?: PlaySound) {
    const [state, setState] = useState<QuizState>({phase: 'idle'})

    // Held in a ref and never depended on, like `useSound` intends: a settings toggle must not
    // resubscribe the feed. It is absent in tests and wherever sound is not wired.
    const play = useRef(playSound)
    useEffect(() => {
        play.current = playSound
    }, [playSound])

    // The country is read when the answer is sent, not when the banner arrived: a player who
    // switched flags mid-question wins it for the flag they are playing now.
    const country = useRef(countryCode)
    useEffect(() => {
        country.current = countryCode
    }, [countryCode])

    // What is on screen, for the callbacks to check against without depending on it — a callback
    // rebuilt on every phase would restart the feed subscription with it.
    const current = useRef(state)
    useEffect(() => {
        current.current = state
    }, [state])

    const dismiss = useCallback(() => setState({phase: 'idle'}), [])

    // A banner arrives only into an empty screen. A question already running is five seconds
    // somebody is in the middle of; the server will not offer a second anyway, and if a stale one
    // did arrive it would take the clock away from under them.
    useEffect(() => {
        if (!master) return

        return master.listenForQuizzes((offer) => {
            if (current.current.phase !== 'idle') return
            setState({phase: 'offered', offer})
            // The banner is easy to miss: it is at the top of the screen and the player is looking
            // at the globe. This is the same reason a bonus box gets a sound when it spawns.
            play.current?.("quiz")
        })
    }, [master])

    // The banner lapses on its own. It is the one phase with no clock drawn on screen: an
    // invitation that quietly goes away reads better than one counting down at you.
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

    // The result clears itself. Nothing waits on it: the charge is already held and the inventory
    // has already been told.
    useEffect(() => {
        if (state.phase !== 'answered') return

        const timer = setTimeout(dismiss, RESULT_MS)
        return () => clearTimeout(timer)
    }, [state, dismiss])

    const open = useCallback(() => {
        const showing = current.current
        if (!master || showing.phase !== 'offered') return

        const {offer} = showing
        // Set before the call, so a second press on the banner does nothing.
        current.current = {phase: 'opening', offer}
        setState(current.current)

        master.openQuiz(offer.token)
            .then((question) => setState((now) =>
                now.phase === 'opening' && now.offer.token === offer.token
                    ? {phase: 'asking', token: offer.token, question}
                    : now))
            .catch((error) => {
                // A banner that lapsed while it was being pressed is ordinary; anything else is a
                // fault worth seeing in the console. Either way the screen clears.
                if (!(error instanceof BonusLostError)) console.error("could not open the quiz", error)
                setState((now) => now.phase === 'opening' && now.offer.token === offer.token ? {phase: 'idle'} : now)
            })
    }, [master])

    /**
     * Sends an answer and shows what it was worth.
     *
     * A `choice` past the end of the three is how **running out of time** is sent, and it is
     * deliberate rather than a hack: the server reads it as wrong — which it is — and answers with
     * the right one, so a question nobody managed to answer still says what it was. There is no
     * other way to learn that, because the bank never leaves the server.
     */
    const answer = useCallback((choice: number) => {
        const showing = current.current
        if (!master || showing.phase !== 'asking') return

        const {token, question} = showing
        const chosen = choice < question.choices.length ? choice : undefined

        // Held here so a second press, or the timer firing on a question already answered, sends
        // nothing: the token is spent by the first answer that lands.
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
