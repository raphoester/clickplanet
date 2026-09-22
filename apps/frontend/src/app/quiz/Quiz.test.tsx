// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, fireEvent, render, screen} from "@testing-library/react"
import {useState} from "react"

import {BonusLostError, QuizMaster} from "../../backends/backend.ts"
import {QuizOffer, QuizOutcome, QuizQuestion} from "../../domain/quiz.ts"
import Quiz from "./Quiz.tsx"
import {RESULT_MS, useQuiz} from "./useQuiz.ts"

const BANNER_MS = 20_000
const ANSWER_MS = 5_000

const QUESTION: QuizQuestion = {
    text: "What is the capital of Estonia?",
    choices: ["Riga", "Tallinn", "Vilnius"],
    deadline: 0,
    window: ANSWER_MS,
}

/**
 * A backend whose whole job is to let a test push a banner in and see what comes back out. The
 * real one is PlanetBackend; the contract between them is the QuizMaster interface.
 */
class FakeMaster implements QuizMaster {
    private listeners: ((offer: QuizOffer) => void)[] = []

    public opened: string[] = []
    public answered: {token: string, choice: number, countryId: string}[] = []

    // What the next call does. Set by the test before it presses anything.
    public open: (token: string) => Promise<QuizQuestion> = () =>
        Promise.resolve({...QUESTION, deadline: performance.now() + ANSWER_MS})

    public outcome: QuizOutcome = {correct: true, correctChoice: 1, reward: {kind: "refill"}}

    listenForQuizzes(onOffered: (offer: QuizOffer) => void): () => void {
        this.listeners.push(onOffered)
        return () => {
            this.listeners = this.listeners.filter((listener) => listener !== onOffered)
        }
    }

    openQuiz(token: string): Promise<QuizQuestion> {
        this.opened.push(token)
        return this.open(token)
    }

    answerQuiz(token: string, choice: number, countryId: string): Promise<QuizOutcome> {
        this.answered.push({token, choice, countryId})
        return Promise.resolve(this.outcome)
    }

    /** A banner from the server. */
    offer(token = "t1") {
        const offer: QuizOffer = {token, expiresAt: performance.now() + BANNER_MS}
        this.listeners.forEach((listener) => listener(offer))
    }

    listening(): boolean {
        return this.listeners.length > 0
    }
}

// The two pieces as they are actually put together in Viewer: the hook drives, the component draws.
function Harness({master, country = "bg"}: {master: QuizMaster, country?: string}) {
    const quiz = useQuiz(master, country)
    return <Quiz state={quiz.state} onOpen={quiz.open} onAnswer={quiz.answer}/>
}

// A promise the fakes resolve settles on the microtask queue, which fake timers do not turn.
async function settle() {
    await act(async () => {
        await Promise.resolve()
        await Promise.resolve()
    })
}

describe("Quiz", () => {
    let master: FakeMaster

    beforeEach(() => {
        vi.useFakeTimers({shouldAdvanceTime: true})
        master = new FakeMaster()
    })

    afterEach(() => {
        cleanup()
        vi.useRealTimers()
    })

    it("shows nothing until the server offers one", () => {
        render(<Harness master={master}/>)

        expect(screen.queryByRole("button")).toBeNull()
    })

    it("gives nothing away about the question it is offering", () => {
        // Not the text, not the choices, and not what it is about. The banner named the subject
        // country and flew its flag once, and that flag was the answer to 417 of the bank's 1014
        // questions — "Estonia" answers "Tallinn is the capital of which country?" on its own, and
        // the flag beside "which of these has the most people?" is the whole question. Picking
        // safer templates is not the fix: a teaser checked against the bank leaks again the first
        // time a template is added.
        render(<Harness master={master}/>)
        act(() => master.offer())

        const banner = screen.getByRole("button")
        expect(banner.textContent).not.toMatch(/Estonia|Tallinn|Riga|Vilnius/)
        expect(banner.querySelector(".country-flag")).toBeNull()
        expect(screen.queryByText(QUESTION.text)).toBeNull()
    })

    it("reads the question only when the banner is pressed", async () => {
        render(<Harness master={master}/>)
        act(() => master.offer())

        expect(master.opened).toEqual([])

        fireEvent.click(screen.getByRole("button"))
        await settle()

        expect(master.opened).toEqual(["t1"])
        expect(screen.getByText(QUESTION.text)).toBeTruthy()
        QUESTION.choices.forEach((choice) => expect(screen.getByRole("button", {name: choice})).toBeTruthy())
    })

    it("opens once however many times the banner is pressed", async () => {
        render(<Harness master={master}/>)
        act(() => master.offer())

        const banner = screen.getByRole("button")
        fireEvent.click(banner)
        fireEvent.click(banner)
        await settle()

        expect(master.opened).toEqual(["t1"])
    })

    it("takes the banner away by itself if nobody presses it", () => {
        render(<Harness master={master}/>)
        act(() => master.offer())

        act(() => void vi.advanceTimersByTime(BANNER_MS + 100))

        expect(screen.queryByRole("button")).toBeNull()
    })

    it("sends the answer for the flag the player is on now", async () => {
        render(<Harness master={master} country="bg"/>)
        act(() => master.offer())

        fireEvent.click(screen.getByRole("button"))
        await settle()

        fireEvent.click(screen.getByRole("button", {name: "Tallinn"}))
        await settle()

        expect(master.answered).toEqual([{token: "t1", choice: 1, countryId: "bg"}])
    })

    it("says what a right answer won", async () => {
        render(<Harness master={master}/>)
        act(() => master.offer())
        fireEvent.click(screen.getByRole("button"))
        await settle()

        fireEvent.click(screen.getByRole("button", {name: "Tallinn"}))
        await settle()

        expect(screen.getByText("Correct")).toBeTruthy()
        expect(screen.getByText("Refill")).toBeTruthy()
    })

    it("says which one was right when it was not the one pressed", async () => {
        master.outcome = {correct: false, correctChoice: 1}

        render(<Harness master={master}/>)
        act(() => master.offer())
        fireEvent.click(screen.getByRole("button"))
        await settle()

        fireEvent.click(screen.getByRole("button", {name: "Riga"}))
        await settle()

        expect(screen.getByText("Not quite")).toBeTruthy()
        expect(screen.getByText("Tallinn")).toBeTruthy()
    })

    it("answers nothing, and learns the answer, when the time runs out", async () => {
        master.outcome = {correct: false, correctChoice: 1}

        render(<Harness master={master}/>)
        act(() => master.offer())
        fireEvent.click(screen.getByRole("button"))
        await settle()

        await act(async () => {
            vi.advanceTimersByTime(ANSWER_MS + 100)
            await Promise.resolve()
        })
        await settle()

        // A choice past the end of the three: the server reads it as wrong, which it is, and says
        // which one was right. There is no other way to learn it — the bank never leaves the server.
        expect(master.answered).toEqual([{token: "t1", choice: 3, countryId: "bg"}])
        expect(screen.getByText("Out of time")).toBeTruthy()
        expect(screen.getByText("Tallinn")).toBeTruthy()
    })

    it("answers once, however many choices are pressed", async () => {
        render(<Harness master={master}/>)
        act(() => master.offer())
        fireEvent.click(screen.getByRole("button"))
        await settle()

        fireEvent.click(screen.getByRole("button", {name: "Riga"}))
        fireEvent.click(screen.getByRole("button", {name: "Tallinn"}))
        await settle()

        expect(master.answered).toHaveLength(1)
    })

    it("does not answer again when the clock runs out on a question already answered", async () => {
        render(<Harness master={master}/>)
        act(() => master.offer())
        fireEvent.click(screen.getByRole("button"))
        await settle()

        fireEvent.click(screen.getByRole("button", {name: "Tallinn"}))
        await settle()

        act(() => void vi.advanceTimersByTime(ANSWER_MS + 100))

        expect(master.answered).toHaveLength(1)
    })

    it("clears the result by itself", async () => {
        render(<Harness master={master}/>)
        act(() => master.offer())
        fireEvent.click(screen.getByRole("button"))
        await settle()

        fireEvent.click(screen.getByRole("button", {name: "Tallinn"}))
        await settle()

        act(() => void vi.advanceTimersByTime(RESULT_MS + 100))

        expect(screen.queryByText("Correct")).toBeNull()
    })

    it("leaves nothing on screen when the banner got away before it opened", async () => {
        master.open = () => Promise.reject(new BonusLostError())

        render(<Harness master={master}/>)
        act(() => master.offer())
        fireEvent.click(screen.getByRole("button"))
        await settle()

        expect(screen.queryByRole("button")).toBeNull()
        expect(screen.queryByRole("dialog")).toBeNull()
    })

    it("does not replace a question already running with a new banner", async () => {
        // The server will not offer a second, but a stale one arriving would take the clock away
        // from under somebody mid-answer.
        render(<Harness master={master}/>)
        act(() => master.offer("t1"))
        fireEvent.click(screen.getByRole("button"))
        await settle()

        act(() => master.offer("t2"))

        expect(screen.getByText(QUESTION.text)).toBeTruthy()
        expect(screen.queryByRole("button", {name: /Answer a question/})).toBeNull()
    })

    it("stops listening when it goes away", () => {
        const {unmount} = render(<Harness master={master}/>)
        expect(master.listening()).toBe(true)

        unmount()

        expect(master.listening()).toBe(false)
    })

    it("makes room at the top by saying it is there", async () => {
        // BombNews reads this: both want the band at the top, and the bomb line is the one that
        // gives it up. The assertion is really about the phase a caller can see.
        render(<Harness master={master}/>)
        expect(screen.queryByRole("dialog")).toBeNull()

        act(() => master.offer())
        expect(screen.getByRole("button")).toBeTruthy()

        fireEvent.click(screen.getByRole("button"))
        await settle()

        expect(screen.getByRole("dialog")).toBeTruthy()
    })

    it("shows nothing at all for a backend that asks no questions", () => {
        function NoMaster() {
            const [country] = useState("bg")
            const quiz = useQuiz(undefined, country)
            return <Quiz state={quiz.state} onOpen={quiz.open} onAnswer={quiz.answer}/>
        }

        render(<NoMaster/>)

        expect(screen.queryByRole("button")).toBeNull()
    })
})
