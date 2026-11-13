import {CSSProperties, ReactNode, useEffect, useRef, useState} from 'react'
import {BonusReward} from '../../domain/bonus.ts'
import {QuizOutcome, QuizQuestion, secondsLeft, timeLeft, untilNextSecond} from '../../domain/quiz.ts'
import {now as budgetNow} from '../../backends/clickBudget.ts'
import {t, translator} from '../../i18n/page.ts'
import {rich} from '../../i18n/rich.tsx'
import {Key} from '../../i18n/translator.ts'
import {describeReward} from '../bonus/describeReward.ts'
import BonusIcon from '../components/BonusIcon.tsx'
import {CloseIcon, PlayIcon} from '../components/icons.tsx'
import {useEdge} from '../components/useEdge.ts'
import {useQuiz} from './quizStore.ts'
import './Quiz.css'

export const QUIZ_BOTTOM = "--quiz-bottom"

type Frame = (element: HTMLDivElement | null) => void

const PRIZES: BonusReward["kind"][] = ["refill", "bomb", "spreadClicks", "encloseClicks", "shields"]

const KEYS = ["A", "B", "C", "D", "E"]

const HURRY_S = 3

const CONFETTI: CSSProperties[] = Array.from({length: 14}, (_, index) => {
    const angle = index / 14 * 2 * Math.PI
    const reach = 48 + (index % 3) * 16
    return {
        "--x": `${Math.round(Math.cos(angle) * reach)}px`,
        "--y": `${Math.round(Math.sin(angle) * reach)}px`,
        "--spin": `${index * 67 % 360}deg`,
    } as CSSProperties
})

export default function Quiz() {
    const state = useQuiz((quiz) => quiz.state)
    const open = useQuiz((quiz) => quiz.open)
    const answer = useQuiz((quiz) => quiz.answer)
    const forfeit = useQuiz((quiz) => quiz.forfeit)
    const dismiss = useQuiz((quiz) => quiz.dismiss)

    const [frame, setFrame] = useState<HTMLDivElement | null>(null)
    useEdge(frame, QUIZ_BOTTOM, "bottom")

    switch (state.phase) {
        case 'idle':
            return null
        case 'offered':
        case 'opening':
            return <Banner frame={setFrame} working={state.phase === 'opening'} onOpen={open}/>
        case 'asking':
        case 'answering':
            return <Question
                key={state.token}
                frame={setFrame}
                question={state.question}
                sent={state.phase === 'answering'}
                onAnswer={answer}
                onLeave={forfeit}
            />
        case 'answered':
            return <Result
                frame={setFrame}
                question={state.question}
                outcome={state.outcome}
                awaiting={state.awaiting}
                chosen={state.chosen}
                left={state.left}
                onClose={dismiss}
            />
    }
}

function Banner({frame, working, onOpen}: {frame: Frame, working: boolean, onOpen: () => void}) {
    return <div ref={frame} className="quiz quiz--banner">
        <button
            type="button"
            className="quiz-banner panel"
            onClick={onOpen}
            disabled={working}
            aria-label={t("quiz.banner.label")}
        >
            <span className="quiz-banner-shine" aria-hidden="true"/>
            <PrizeReel/>
            <span className="quiz-banner-words">
                <strong className="quiz-banner-title">{t("quiz.banner.title")}</strong>
                <span className="quiz-banner-detail">{t("quiz.banner.detail")}</span>
            </span>
            <span className="quiz-banner-go" aria-hidden="true"><PlayIcon size={16}/></span>
            <Sparkle place={1}/>
            <Sparkle place={2}/>
            <Sparkle place={3}/>
        </button>
    </div>
}

function Question({frame, question, sent, onAnswer, onLeave}: {
    frame: Frame,
    question: QuizQuestion,
    sent: boolean,
    onAnswer: (choice: number) => void,
    onLeave: () => void,
}) {
    const [picked, setPicked] = useState<number>()
    const seconds = useSecondsLeft(question, picked === undefined)
    const [clock] = useState(() => clockStyle(question, budgetNow()))

    const answer = useRef(onAnswer)
    const leave = useRef(onLeave)
    useEffect(() => {
        answer.current = onAnswer
        leave.current = onLeave
    }, [onAnswer, onLeave])

    useEffect(() => {
        const timer = setTimeout(
            () => answer.current(question.choices.length),
            Math.max(0, question.deadline - budgetNow()),
        )
        return () => clearTimeout(timer)
    }, [question])

    useEffect(() => leaving(() => leave.current()), [])

    const closed = picked !== undefined || seconds === 0 || sent
    const pick = (index: number) => {
        if (closed) return
        setPicked(index)
        answer.current(index)
    }

    const card = ["quiz-card", "panel"]
    if (picked !== undefined) card.push("quiz-card--picked")
    else if (seconds <= HURRY_S) card.push("quiz-card--hurry")

    return <div ref={frame} className="quiz quiz--asking" role="dialog" aria-label={t("quiz.label")}>
        <div className={card.join(" ")}>
            <div className="quiz-head">
                <PrizeReel/>
                <div className="quiz-clock" aria-hidden="true">
                    <div className="quiz-clock-bar" style={clock}/>
                </div>
                <span key={seconds} className="quiz-seconds" aria-hidden="true">{translator().number(seconds)}</span>
            </div>

            <p className="quiz-question">{question.text}</p>

            <div className="quiz-choices">
                {question.choices.map((choice, index) =>
                    <button
                        type="button"
                        key={`${index}-${choice}`}
                        className={`button button-secondary quiz-choice${!closed ? "" : index === picked ? " quiz-choice--picked" : " quiz-choice--dim"}`}
                        style={{"--choice": index} as CSSProperties}
                        disabled={closed}
                        onClick={() => pick(index)}
                    >
                        <span className="quiz-choice-key" aria-hidden="true">{KEYS[index]}</span>
                        <span className="quiz-choice-text">{choice}</span>
                    </button>)}
            </div>
        </div>
    </div>
}

function Result({frame, question, outcome, awaiting, chosen, left, onClose}: {
    frame: Frame,
    question: QuizQuestion,
    outcome: QuizOutcome,
    awaiting?: boolean,
    chosen?: number,
    left?: boolean,
    onClose: () => void,
}) {
    const won = outcome.reward
    const reward = won && describeReward(won)

    return <div ref={frame} className={`quiz quiz--result quiz--${outcome.correct ? "right" : "wrong"}`} role="status" aria-live="polite">
        <div className="quiz-card panel">
            <div className="quiz-head quiz-head--verdict">
                {won ? <PrizeWon kind={won.kind}/> : awaiting && <PrizeReel/>}
                <div className="quiz-verdict-words">
                    <p className="quiz-verdict">
                        {t(verdict(outcome, chosen, left))}
                    </p>
                    {won && reward && <p className={`quiz-reward quiz-kind--${won.kind}`}>{reward.title}</p>}
                    {reward?.detail && <p className="quiz-reward-detail">{reward.detail}</p>}
                    {outcome.correct && !reward && !awaiting &&
                        <p className="quiz-reward-detail">{t("quiz.inventoryFull")}</p>}
                </div>
                <button type="button" className="icon-button quiz-close" aria-label={t("common.close")} onClick={onClose}>
                    <CloseIcon/>
                </button>
            </div>

            <p className="quiz-question">{question.text}</p>

            <ol className="quiz-choices">
                {question.choices.map((choice, index) => {
                    const mark = index === outcome.correctChoice ? "right" : index === chosen ? "wrong" : "other"
                    return <li key={`${index}-${choice}`} className={`button button-secondary quiz-choice quiz-choice--${mark}`}>
                        <span className="quiz-choice-key" aria-hidden="true">
                            {mark === "right" ? <CheckMark/> : mark === "wrong" ? <CrossMark/> : KEYS[index]}
                        </span>
                        {choiceLine(mark, choice)}
                    </li>
                })}
            </ol>
        </div>
    </div>
}

function verdict(outcome: QuizOutcome, chosen?: number, left?: boolean): Key {
    if (outcome.correct) return "quiz.correct"
    if (left) return "quiz.left"
    return chosen === undefined ? "quiz.outOfTime" : "quiz.notQuite"
}

// Calls back when the page loses the player while a question is open: another window took the focus (the phone's
// assistant or screen search drawn over the page, another app, a click outside the browser) or the tab was hidden.
// A question that can be read off the screen by something else is no longer answered from what the player knows.
function leaving(callback: () => void): () => void {
    const blurred = (event: FocusEvent) => {
        if (!(event.target instanceof Node)) callback()
    }
    const hidden = () => {
        if (document.visibilityState === 'hidden') callback()
    }
    window.addEventListener('blur', blurred)
    document.addEventListener('visibilitychange', hidden)
    return () => {
        window.removeEventListener('blur', blurred)
        document.removeEventListener('visibilitychange', hidden)
    }
}

function choiceLine(mark: "right" | "wrong" | "other", choice: string): ReactNode {
    const text = <span className="quiz-choice-text">{choice}</span>
    const hidden = (words: ReactNode[]) => <span className="sr-only">{words}</span>
    switch (mark) {
        case "right":
            return rich("quiz.theAnswer", {choice: text, hidden})
        case "wrong":
            return rich("quiz.yourPick", {choice: text, hidden})
        case "other":
            return text
    }
}

function PrizeReel() {
    return <span className="quiz-prize" aria-hidden="true">
        <span className="quiz-prize-window">
            {PRIZES.map((kind, index) =>
                <span key={kind} className={`quiz-prize-face quiz-kind--${kind}`} style={{"--face": index} as CSSProperties}>
                    <BonusIcon kind={kind}/>
                </span>)}
        </span>
        <span className="quiz-prize-tag">?</span>
    </span>
}

function PrizeWon({kind}: {kind: BonusReward["kind"]}) {
    return <span className={`quiz-prize quiz-prize--won quiz-kind--${kind}`} aria-hidden="true">
        <span className="quiz-prize-rays"/>
        <span className="quiz-prize-window">
            <span className="quiz-prize-face"><BonusIcon kind={kind}/></span>
        </span>
        <span className="quiz-confetti">
            {CONFETTI.map((style, index) => <span key={index} style={style}/>)}
        </span>
    </span>
}

function Sparkle({place}: {place: number}) {
    return <svg className={`quiz-sparkle quiz-sparkle--${place}`} viewBox="0 0 20 20" aria-hidden="true">
        <polygon points="10,0 12.5,7.5 20,10 12.5,12.5 10,20 7.5,12.5 0,10 7.5,7.5"/>
    </svg>
}

function CheckMark() {
    return <svg className="quiz-mark" viewBox="0 0 20 20" aria-hidden="true">
        <path d="M4 10.5 L8.5 15 L16 5.5"/>
    </svg>
}

function CrossMark() {
    return <svg className="quiz-mark" viewBox="0 0 20 20" aria-hidden="true">
        <path d="M5.5 5.5 L14.5 14.5 M14.5 5.5 L5.5 14.5"/>
    </svg>
}

function clockStyle(question: QuizQuestion, at: number): CSSProperties {
    const spent = (1 - timeLeft(question, at)) * question.window
    return {
        "--quiz-window": `${question.window}ms`,
        "--quiz-spent": `${-Math.round(spent)}ms`,
    } as CSSProperties
}

function useSecondsLeft(question: QuizQuestion, running: boolean): number {
    const [seconds, setSeconds] = useState(() => secondsLeft(question, budgetNow()))

    useEffect(() => {
        if (!running) return

        let timer: ReturnType<typeof setTimeout> | undefined
        const schedule = () => {
            const wait = untilNextSecond(question, budgetNow())
            if (!Number.isFinite(wait)) return
            timer = setTimeout(() => {
                setSeconds(secondsLeft(question, budgetNow()))
                schedule()
            }, wait)
        }
        schedule()

        return () => clearTimeout(timer)
    }, [question, running])

    return seconds
}
