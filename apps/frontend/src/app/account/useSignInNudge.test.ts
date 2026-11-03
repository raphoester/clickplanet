// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, renderHook} from "@testing-library/react"
import {StrictMode} from "react"
import {useQuiz} from "../quiz/quizStore.ts"
import {NUDGE_AFTER_MS, useSignInNudge} from "./useSignInNudge.ts"

type Props = {offered: boolean, dryTaps: number}

function nudge(initialProps: Props = {offered: true, dryTaps: 0}, wrapper?: typeof StrictMode) {
    return renderHook((props: Props) => useSignInNudge(props.offered, props.dryTaps), {initialProps, wrapper})
}

const wait = (ms: number) => act(() => {
    vi.advanceTimersByTime(ms)
})

const quizUp = (up: boolean) => act(() => {
    useQuiz.setState({state: up ? {phase: "offered", offer: {token: "t1", expiresAt: Infinity}} : {phase: "idle"}})
})

beforeEach(() => {
    vi.useFakeTimers()
})

afterEach(() => {
    useQuiz.setState({state: {phase: "idle"}})
    vi.useRealTimers()
})

describe("useSignInNudge", () => {
    it("opens a moment after a dry tap", async () => {
        const {result, rerender} = nudge()
        expect(result.current.open).toBe(false)

        rerender({offered: true, dryTaps: 1})
        await wait(NUDGE_AFTER_MS - 1)
        expect(result.current.open).toBe(false)
        await wait(1)

        expect(result.current.open).toBe(true)
    })

    it("opens in StrictMode too", async () => {
        const {result, rerender} = nudge(undefined, StrictMode)

        rerender({offered: true, dryTaps: 1})
        await wait(NUDGE_AFTER_MS)

        expect(result.current.open).toBe(true)
    })

    it("never opens over a quiz, and a quiz in the wait puts it off to the next dry tap", async () => {
        const {result, rerender} = nudge()

        quizUp(true)
        rerender({offered: true, dryTaps: 1})
        await wait(NUDGE_AFTER_MS)
        expect(result.current.open).toBe(false)

        quizUp(false)
        rerender({offered: true, dryTaps: 2})
        await wait(NUDGE_AFTER_MS / 2)
        quizUp(true)
        await wait(NUDGE_AFTER_MS)
        expect(result.current.open).toBe(false)

        quizUp(false)
        await wait(NUDGE_AFTER_MS)
        expect(result.current.open, "the quiz closing is no dry tap").toBe(false)

        rerender({offered: true, dryTaps: 3})
        await wait(NUDGE_AFTER_MS)
        expect(result.current.open).toBe(true)
    })

    it("opens once a page", async () => {
        const {result, rerender} = nudge()
        rerender({offered: true, dryTaps: 1})
        await wait(NUDGE_AFTER_MS)

        act(() => result.current.close())
        rerender({offered: true, dryTaps: 2})
        await wait(NUDGE_AFTER_MS)

        expect(result.current.open).toBe(false)
    })

    it("does not open for a player it is not offered to", async () => {
        const {result, rerender} = nudge({offered: false, dryTaps: 0})

        rerender({offered: false, dryTaps: 1})
        await wait(NUDGE_AFTER_MS)
        expect(result.current.open).toBe(false)

        rerender({offered: true, dryTaps: 1})
        await wait(NUDGE_AFTER_MS)
        expect(result.current.open, "being offered it is no dry tap").toBe(false)
    })
})
