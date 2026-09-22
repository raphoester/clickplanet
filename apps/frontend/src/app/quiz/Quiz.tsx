import {useEffect, useRef, useState} from 'react'
import {describeReward} from '../../domain/bonus.ts'
import {QuizOutcome, QuizQuestion, timeLeft} from '../../domain/quiz.ts'
import {now as budgetNow} from '../../backends/clickBudget.ts'
import BonusIcon from '../components/BonusIcon.tsx'
import {QuizState} from './useQuiz.ts'
import './Quiz.css'

export type QuizProps = {
    state: QuizState
    /** Opens the banner, which is what starts the clock. */
    onOpen: () => void
    /** The choice pressed, or `choices.length` for the time running out. */
    onAnswer: (choice: number) => void
}

/**
 * The quiz, at the top of the screen: the banner, the question, and what the answer was worth.
 *
 * One component for all three because they are one thing in one place — the card grows out of the
 * banner rather than appearing somewhere else — and because only one of them is ever on screen.
 */
export default function Quiz({state, onOpen, onAnswer}: QuizProps) {
    switch (state.phase) {
        case 'idle':
            return null
        case 'offered':
        case 'opening':
            return <Banner working={state.phase === 'opening'} onOpen={onOpen}/>
        case 'asking':
            return <Question question={state.question} onAnswer={onAnswer}/>
        case 'answered':
            return <Result question={state.question} outcome={state.outcome} chosen={state.chosen}/>
    }
}

/**
 * The invitation, and it gives nothing away — not the question, not the choices, and not what it
 * is about.
 *
 * It named the subject country and flew its flag once. That flag *was* the answer to 417 of the
 * bank's 1014 questions: every "Tallinn is the capital of which country?" and every "which of
 * these has the most people?" is answered by the flag beside it. The fix is not to pick safer
 * templates — a teaser that has to be checked against every question in the bank leaks again the
 * first time a template is added — it is to say nothing. What makes the banner worth pressing is
 * the charge behind it, not a hint.
 *
 * It draws no countdown of its own either: it is free to ignore, and a clock ticking at you over
 * the planet would say otherwise. It goes away by itself.
 */
function Banner({working, onOpen}: {working: boolean, onOpen: () => void}) {
    return <div className="quiz quiz--banner">
        <button
            type="button"
            className="quiz-banner"
            onClick={onOpen}
            disabled={working}
            aria-label="Answer a question for a bonus"
        >
            <span className="quiz-banner-mark" aria-hidden="true">?</span>
            <span className="quiz-banner-words">
                <strong className="quiz-banner-title">Quiz</strong>
                <span className="quiz-banner-detail">Answer one for a bonus — 5 seconds</span>
            </span>
        </button>
    </div>
}

/**
 * The question and its three choices, with the clock already running.
 *
 * **The bar starts at what is actually left, not at full.** The five seconds are the server's and
 * they began when it answered, so a slow round trip has already spent some of them: a bar that
 * started full here would promise time the player does not have. From there it is one CSS
 * transition to empty — cheaper than a frame loop beside a WebGL globe, and smooth for the same
 * reason.
 */
function Question({question, onAnswer}: {question: QuizQuestion, onAnswer: (choice: number) => void}) {
    const [left, setLeft] = useState(() => timeLeft(question, budgetNow()))
    const bar = useRef<HTMLDivElement>(null)

    // Held in a ref rather than depended on, for the reason BonusAward holds its: this sits over a
    // leaderboard that republishes twice a second, and a rebuilt callback in the dependencies
    // would restart the countdown on every one of those renders.
    const answer = useRef(onAnswer)
    useEffect(() => {
        answer.current = onAnswer
    }, [onAnswer])

    useEffect(() => {
        const remaining = Math.max(0, question.deadline - budgetNow())

        // Painted at the fraction that is left, then told to run to empty over what remains. Two
        // frames, because a transition set in the same frame as the value it starts from has
        // nothing to animate away from.
        setLeft(timeLeft(question, budgetNow()))
        const frame = requestAnimationFrame(() => requestAnimationFrame(() => {
            const element = bar.current
            if (!element) return
            element.style.transition = `transform ${remaining}ms linear`
            element.style.transform = "scaleX(0)"
        }))

        // Out of time is sent as a choice past the end of the three: the server reads it as wrong,
        // which it is, and says which one was right — see useQuiz.
        const timer = setTimeout(() => answer.current(question.choices.length), remaining)

        return () => {
            cancelAnimationFrame(frame)
            clearTimeout(timer)
        }
    }, [question])

    return <div className="quiz quiz--asking" role="dialog" aria-label="Quiz">
        <div className="quiz-card">
            <div className="quiz-clock" aria-hidden="true">
                <div className="quiz-clock-bar" ref={bar} style={{transform: `scaleX(${left})`}}/>
            </div>

            <p className="quiz-question">{question.text}</p>

            <div className="quiz-choices">
                {question.choices.map((choice, index) =>
                    <button
                        type="button"
                        key={`${index}-${choice}`}
                        className="quiz-choice"
                        onClick={() => answer.current(index)}
                    >{choice}</button>)}
            </div>
        </div>
    </div>
}

/**
 * What the answer was worth.
 *
 * It says which one was right whether or not that was the one pressed. A wrong answer costs
 * nothing — the whole reason a guess is safe is that there is nothing to lose — so the only thing
 * left to give back is the answer itself.
 */
function Result({question, outcome, chosen}: {question: QuizQuestion, outcome: QuizOutcome, chosen?: number}) {
    const right = question.choices[outcome.correctChoice]
    const reward = outcome.reward && describeReward(outcome.reward)

    return <div className={`quiz quiz--result quiz--${outcome.correct ? "right" : "wrong"}`} role="status" aria-live="polite">
        <div className="quiz-card">
            <p className="quiz-verdict">
                {outcome.correct ? "Correct" : chosen === undefined ? "Out of time" : "Not quite"}
            </p>

            {!outcome.correct && right !== undefined &&
                <p className="quiz-answer">The answer was <strong>{right}</strong></p>}

            {reward
                ? <p className={`quiz-reward quiz-reward--${outcome.reward?.kind}`}>
                    <span className="quiz-reward-icon" aria-hidden="true">
                        {outcome.reward && <BonusIcon kind={outcome.reward.kind}/>}
                    </span>
                    <strong>{reward.title}</strong>
                    <span className="quiz-reward-detail">{reward.detail}</span>
                </p>
                // Right, and nothing to give: the player already holds one of everything a quiz
                // pays out in. Saying so is better than a blank card that looks like a bug.
                : outcome.correct && <p className="quiz-answer">Your inventory is already full</p>}
        </div>
    </div>
}
