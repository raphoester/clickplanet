import {useEffect, useRef, useState} from 'react'
import {describeReward} from '../../domain/bonus.ts'
import {QuizOutcome, QuizQuestion, timeLeft} from '../../domain/quiz.ts'
import {now as budgetNow} from '../../backends/clickBudget.ts'
import BonusIcon from '../components/BonusIcon.tsx'
import {QuizState} from './useQuiz.ts'
import './Quiz.css'

export type QuizProps = {
    state: QuizState
    onOpen: () => void
    onAnswer: (choice: number) => void
}

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

// The "8 seconds" below must match the backend's bonus.quiz.answerWindow.
function Banner({working, onOpen}: {working: boolean, onOpen: () => void}) {
    return <div className="quiz quiz--banner">
        <button
            type="button"
            className="quiz-banner panel"
            onClick={onOpen}
            disabled={working}
            aria-label="Answer a question for a bonus"
        >
            <span className="quiz-banner-mark" aria-hidden="true">?</span>
            <span className="quiz-banner-words">
                <strong className="quiz-banner-title">Quiz</strong>
                <span className="quiz-banner-detail">Answer one for a bonus — 8 seconds</span>
            </span>
        </button>
    </div>
}

function Question({question, onAnswer}: {question: QuizQuestion, onAnswer: (choice: number) => void}) {
    const [left, setLeft] = useState(() => timeLeft(question, budgetNow()))
    const bar = useRef<HTMLDivElement>(null)

    const answer = useRef(onAnswer)
    useEffect(() => {
        answer.current = onAnswer
    }, [onAnswer])

    useEffect(() => {
        const remaining = Math.max(0, question.deadline - budgetNow())

        setLeft(timeLeft(question, budgetNow()))
        // Two frames: a transition set in the frame that paints its start value does not run.
        const frame = requestAnimationFrame(() => requestAnimationFrame(() => {
            const element = bar.current
            if (!element) return
            element.style.transition = `transform ${remaining}ms linear`
            element.style.transform = "scaleX(0)"
        }))

        const timer = setTimeout(() => answer.current(question.choices.length), remaining)

        return () => {
            cancelAnimationFrame(frame)
            clearTimeout(timer)
        }
    }, [question])

    return <div className="quiz quiz--asking" role="dialog" aria-label="Quiz">
        <div className="quiz-card panel">
            <div className="quiz-clock" aria-hidden="true">
                <div className="quiz-clock-bar" ref={bar} style={{transform: `scaleX(${left})`}}/>
            </div>

            <p className="quiz-question">{question.text}</p>

            <div className="quiz-choices">
                {question.choices.map((choice, index) =>
                    <button
                        type="button"
                        key={`${index}-${choice}`}
                        className="button"
                        onClick={() => answer.current(index)}
                    >{choice}</button>)}
            </div>
        </div>
    </div>
}

function Result({question, outcome, chosen}: {question: QuizQuestion, outcome: QuizOutcome, chosen?: number}) {
    const right = question.choices[outcome.correctChoice]
    const reward = outcome.reward && describeReward(outcome.reward)

    return <div className={`quiz quiz--result quiz--${outcome.correct ? "right" : "wrong"}`} role="status" aria-live="polite">
        <div className="quiz-card panel">
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
                    {reward.detail && <span className="quiz-reward-detail">{reward.detail}</span>}
                </p>
                : outcome.correct && <p className="quiz-answer">Your inventory is already full</p>}
        </div>
    </div>
}
